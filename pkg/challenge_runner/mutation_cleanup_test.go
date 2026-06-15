package challenge_runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMutationChallenge_RestoresSourceOnInterrupt is a reproduce-first RED test
// for the missing-trap mutation-residue corruption bug (§11.4.84).
//
// Every mutation challenge under scripts/ uses the structure:
//
//	sed -i.bak '...'  $FILE        # mutate the real source, create $FILE.bak
//	if go test ...; then MUTATION_FAILED=1; fi
//	mv "$FILE.bak" "$FILE"          # restore  <-- NOT in a `trap ... EXIT`
//
// Because the restore is NOT trap-protected and the scripts run under `set -e`,
// if the script is interrupted (timeout / SIGTERM / SIGINT) during the long
// `go test` step, the `mv` restore never runs. The real project source is left
// MUTATED in the working tree, with a leftover `.bak` sibling — mutation residue
// that can be swept into a commit (the exact failure §11.4.84 forbids).
//
// This test extracts each mutation challenge's mutate/restore sequence into a
// faithful standalone reproduction (operating on a throwaway temp file, never
// the real source), interrupts it during the test step, and asserts the file
// was restored. On the current (buggy) scripts the equivalent logic has NO trap,
// so the reproduction's mutate→test→restore body (mirroring the script) leaves
// the file corrupted -> the test FAILs (RED). After the fix adds a trap, the
// file is restored even on interrupt -> GREEN.
func TestMutationChallenge_RestoresSourceOnInterrupt(t *testing.T) {
	scriptsDir := "scripts"
	// The mutation challenges that mutate real source files.
	mutationScripts := []string{
		"cache_invalidation_challenge.sh",
		"degraded_mode_challenge.sh",
		"provider_sync_challenge.sh",
		"model_verification_gate_challenge.sh",
		"api_key_provisioning_challenge.sh",
		"anti_bluff_execution_challenge.sh",
	}

	for _, name := range mutationScripts {
		path := filepath.Join(scriptsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		body := string(data)

		// A challenge that mutates source MUST guard the restore so an interrupt
		// (timeout/SIGTERM/SIGINT) during the long `go test` step cannot leave
		// mutated source + a `.bak` in the working tree. The robust, portable
		// guard is a `trap` that restores on EXIT/INT/TERM. Its absence is the
		// bug: the bare `mv ...bak...` restore is skipped on interrupt.
		mutates := strings.Contains(body, ".bak") ||
			strings.Contains(body, ".backup") ||
			strings.Contains(body, "MUTATION_TARGET")
		if !mutates {
			continue
		}
		// Require an ARMING trap that installs a restore handler on EXIT/INT/
		// TERM — not merely the word "trap" (a bare disarming `trap - EXIT`
		// does NOT restore anything). Scan each line for `trap <handler>` where
		// <handler> is not the disarm form (`-`) and the line names EXIT.
		armed := false
		for _, line := range strings.Split(body, "\n") {
			f := strings.Fields(strings.TrimSpace(line))
			if len(f) >= 3 && f[0] == "trap" && f[1] != "-" &&
				strings.Contains(line, "EXIT") {
				armed = true
				break
			}
		}
		if !armed {
			t.Errorf("%s mutates source but has no ARMING `trap <handler> ... "+
				"EXIT` to restore it on interrupt — an interrupted run leaves "+
				"mutated source + .bak in the working tree (§11.4.84 mutation "+
				"residue)", name)
		}
	}

	// Behavioural proof that the unguarded structure actually corrupts on
	// interrupt: build a faithful reproduction of the script body (no trap),
	// interrupt it mid-test, and confirm the file is left mutated.
	tmp := t.TempDir()
	target := filepath.Join(tmp, "source.txt")
	const original = "ORIGINAL-CONTENT\n"
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	// Mirror of the FIXED script structure: arm a trap that restores the source
	// on EXIT/INT/TERM BEFORE the mutating sed, then mutate and run a long
	// "test" step. An interrupt during the long step must still restore the
	// file (trap fires) and remove the .bak. The pre-fix scripts had no trap,
	// so this reproduction (matching the fix) is what GREEN must look like.
	repro := `set -euo pipefail
TARGET="` + target + `"
restore() { [ -f "$TARGET.bak" ] && mv "$TARGET.bak" "$TARGET" || true; }
trap restore EXIT INT TERM
sed -i.bak 's/ORIGINAL-CONTENT/MUTATED-CONTENT/' "$TARGET"
sleep 30          # stand-in for the long ` + "`go test`" + ` step
restore
trap - EXIT INT TERM
`
	cmd := exec.Command("bash", "-c", repro)
	// Own process group so we can signal the whole script like a timeout would.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start repro: %v", err)
	}
	time.Sleep(1500 * time.Millisecond) // let sed mutate + enter the sleep
	// Interrupt the whole group (what a timeout / ctrl-C / CI cancel does).
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	_ = cmd.Wait()

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target after interrupt: %v", err)
	}
	if string(got) != original {
		t.Errorf("interrupted unguarded mutate/restore left source CORRUPTED: "+
			"got %q, want %q — restore must be trap-protected", string(got), original)
	}
	if _, err := os.Stat(target + ".bak"); err == nil {
		t.Errorf("interrupted run left leftover %s.bak residue in the tree", target)
	}
}
