// Command model-bridge exposes the LLMsVerifier-obtained STRONGEST verified
// model to operators AND to a Claude Code agent session, out of the box.
//
// Subcommands:
//
//	model-bridge best-model            print the strongest verified model + fallback chain
//	model-bridge list                  print all verified models (JSON) ranked strongest-first
//	model-bridge invoke -prompt "..."  send a raw prompt to the strongest model, print completion
//	model-bridge invoke < file         read the prompt from stdin
//	model-bridge mcp                    run an MCP stdio server (tools: bridge_invoke / bridge_best_model / bridge_list)
//
// Out-of-the-box: reads provider API keys from the environment (*_API_KEY),
// runs the in-process LLMsVerifier pipeline (no running service required),
// selects the strongest verified model, and invokes it. Set LLMSVERIFIER_API_URL
// to use a running LLMsVerifier HTTP service instead.
//
// API-key VALUES are NEVER printed (§11.4.10) — only provider IDs, model IDs,
// scores, and real model completions.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "best-model":
		os.Exit(runBestModel(args))
	case "list":
		os.Exit(runList(args))
	case "invoke":
		os.Exit(runInvoke(args))
	case "mcp":
		os.Exit(runMCP(args))
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `model-bridge — strongest LLMsVerifier-verified model, out of the box

Usage:
  model-bridge best-model [-db PATH] [-min-score F] [-max N] [-force-verify]
  model-bridge list       [-db PATH] [-min-score F] [-max N] [-force-verify]
  model-bridge invoke     [-prompt TEXT] [-system TEXT] [-db PATH] ...   (prompt also read from stdin)
  model-bridge mcp        [-db PATH] ...                                  (MCP stdio server)

Bootstrap:
  Reads provider keys from env (*_API_KEY), runs the in-process verification
  pipeline (no service needed), selects the strongest verified model.
  Set LLMSVERIFIER_API_URL to use a running LLMsVerifier HTTP service instead.
  API-key values are never printed.
`)
}

// commonFlags defines the bootstrap flags shared by every subcommand and returns
// bridge.Options built from them.
func commonFlags(fs *flag.FlagSet) func() bridge.Options {
	db := fs.String("db", "./data/verified_models.db", "SQLite store path for the in-process pipeline")
	minScore := fs.Float64("min-score", 0.0, "minimum overall score (exclusive) for a model to count as verified")
	maxModels := fs.Int("max", 3, "max models per provider to verify when a fresh pass is needed (0 = no cap)")
	timeout := fs.Duration("verify-timeout", 5*time.Minute, "timeout for an in-process verification pass")
	force := fs.Bool("force-verify", false, "always run a fresh verification pass even if the store has models")
	apiURL := fs.String("api-url", "", "LLMsVerifier HTTP service URL (overrides in-process; also LLMSVERIFIER_API_URL)")
	apiKey := fs.String("api-key", "", "Bearer token for the LLMsVerifier HTTP service")
	return func() bridge.Options {
		return bridge.Options{
			APIURL:               *apiURL,
			APIKey:               *apiKey,
			DBPath:               *db,
			MaxModelsPerProvider: *maxModels,
			MinScoreThreshold:    *minScore,
			VerifyTimeout:        *timeout,
			ForceVerify:          *force,
		}
	}
}

func openBridge(opts bridge.Options) (*bridge.Bridge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opts.VerifyTimeout+30*time.Second)
	defer cancel()
	return bridge.Open(ctx, opts)
}

func runBestModel(args []string) int {
	fs := flag.NewFlagSet("best-model", flag.ContinueOnError)
	getOpts := commonFlags(fs)
	jsonOut := fs.Bool("json", false, "emit JSON instead of human-readable text")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	b, err := openBridge(getOpts())
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}

	ctx := context.Background()
	models, err := b.ListVerified(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}
	best := models[0]

	if *jsonOut {
		out := map[string]any{
			"source":         b.Source(),
			"strongest":      best,
			"fallback_chain": fallbackIDs(models),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return 0
	}

	fmt.Printf("source:    %s\n", b.Source())
	fmt.Printf("strongest: %s/%s  (score %.3f, factory=%s)\n", best.ProviderID, best.ModelID, best.Score, best.FactoryName)
	if best.BaseURL != "" {
		fmt.Printf("base_url:  %s\n", best.BaseURL)
	}
	fmt.Printf("fallback chain (strongest-first):\n")
	for _, m := range models {
		fmt.Printf("  %2d. %s/%s  (score %.3f)\n", m.FallbackOrder, m.ProviderID, m.ModelID, m.Score)
	}
	return 0
}

func runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	getOpts := commonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	b, err := openBridge(getOpts())
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}
	models, err := b.ListVerified(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]any{"source": b.Source(), "models": models}); err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: encode: %v\n", err)
		return 1
	}
	return 0
}

func runInvoke(args []string) int {
	fs := flag.NewFlagSet("invoke", flag.ContinueOnError)
	getOpts := commonFlags(fs)
	prompt := fs.String("prompt", "", "the prompt to send (if empty, read from stdin)")
	system := fs.String("system", "", "optional system instruction prepended to the prompt")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	p := *prompt
	if strings.TrimSpace(p) == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "model-bridge: read stdin: %v\n", err)
			return 1
		}
		p = string(data)
	}
	if strings.TrimSpace(p) == "" {
		fmt.Fprintln(os.Stderr, "model-bridge: empty prompt (use -prompt or pipe via stdin)")
		return 2
	}

	b, err := openBridge(getOpts())
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}
	out, err := b.Invoke(context.Background(), *system, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge: %v\n", err)
		return 1
	}
	fmt.Println(out)
	return 0
}

// fallbackIDs flattens ranked models into "provider/model" strings.
func fallbackIDs(models []bridge.ModelInfo) []string {
	ids := make([]string, 0, len(models))
	for _, m := range models {
		ids = append(ids, m.ProviderID+"/"+m.ModelID)
	}
	return ids
}

// taskFromFlags is reserved for future task-requirement flags; the bridge's
// strongest-model selection currently uses the zero TaskRequirements.
var _ = selection.TaskRequirements{}
