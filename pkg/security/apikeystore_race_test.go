package security

// APIKeyStore concurrency adversarial test (§11.4.27 SECURITY / concurrency).
//
// ROOT CAUSE (§11.4.102): APIKeyStore.keys is a plain map[string]APIKeyInfo
// with NO synchronization, while the sibling RateLimiter in this same package
// correctly guards its maps with sync.RWMutex. AddKey/RevokeKey write the map;
// ValidateKey reads it. Concurrent callers therefore race the map, which the Go
// runtime turns into a fatal "concurrent map read and map write" / "concurrent
// map writes" panic — a DoS / availability defect on a security-critical
// surface (API-key validation).
//
// This is a reproduce-first RED test per §11.4.146 / §11.4.115:
//   - On the UNSYNCHRONIZED store it data-races (FAIL under -race) and can
//     fatally panic the test binary.
//   - On the FIXED (mutex-guarded) store it is -race-clean and panic-free.
// The attack the test models: many goroutines validating/adding/revoking API
// keys at once (normal concurrent HTTP traffic against an auth component).

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAdv_APIKeyStore_ConcurrentAccessNoRace hammers AddKey / ValidateKey /
// RevokeKey concurrently. Run with -race: pre-fix this fails (race + possible
// fatal map panic); post-fix it is clean.
func TestAdv_APIKeyStore_ConcurrentAccessNoRace(t *testing.T) {
	store := NewAPIKeyStore()

	// Seed some keys so ValidateKey has hits as well as misses.
	const seeded = 50
	for i := 0; i < seeded; i++ {
		store.AddKey(fmt.Sprintf("seed-%d", i), APIKeyInfo{
			Key:    fmt.Sprintf("seed-%d", i),
			UserID: "u",
			Active: true,
		})
	}

	const goroutines = 64
	const iters = 500
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Writers: AddKey
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				k := fmt.Sprintf("k-%d-%d", g, i)
				store.AddKey(k, APIKeyInfo{Key: k, UserID: "u", Active: true})
			}
		}(g)
	}
	// Writers: RevokeKey
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				store.RevokeKey(fmt.Sprintf("seed-%d", i%seeded))
			}
		}(g)
	}
	// Readers: ValidateKey
	var validated int64
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if _, ok := store.ValidateKey(fmt.Sprintf("seed-%d", i%seeded)); ok {
					atomic.AddInt64(&validated, 1)
				}
			}
		}(g)
	}

	wg.Wait()
	// No assertion on the count is needed: the bug manifests as a -race report
	// or a fatal runtime panic. Reaching here cleanly under -race IS the proof.
	_ = atomic.LoadInt64(&validated)
}

// TestAdv_APIKeyStore_RevokeDeniesValidation — functional-correctness extend
// (§11.4.146 STEP 3). The concurrency fix must not weaken the security check:
// a revoked key MUST fail validation, an expired key MUST fail, an unknown key
// MUST fail, and a live active key MUST pass. This guards against a "fix the
// race by making ValidateKey always return false" (or always true) regression.
func TestAdv_APIKeyStore_RevokeDeniesValidation(t *testing.T) {
	store := NewAPIKeyStore()
	store.AddKey("live", APIKeyInfo{Key: "live", UserID: "u", Active: true})

	// Active key validates.
	if _, ok := store.ValidateKey("live"); !ok {
		t.Fatal("active key should validate (fix must not break the allow path)")
	}
	// Unknown key denied.
	if _, ok := store.ValidateKey("does-not-exist"); ok {
		t.Fatal("SECURITY: unknown key was accepted")
	}
	// Revoked key denied.
	store.RevokeKey("live")
	if _, ok := store.ValidateKey("live"); ok {
		t.Fatal("SECURITY: revoked key still validated — revocation bypass")
	}

	// Expired key denied.
	past := time.Now().Add(-time.Hour)
	store.AddKey("expired", APIKeyInfo{Key: "expired", UserID: "u", Active: true, ExpiresAt: &past})
	if _, ok := store.ValidateKey("expired"); ok {
		t.Fatal("SECURITY: expired key validated")
	}
}

// TestAdv_APIKeyStore_ConcurrentRevokeThenValidateConsistent — under concurrent
// revoke+validate on the SAME key, every ValidateKey result must be a clean
// boolean (true while active, false once revoked) and never a torn/garbage
// read. Run with -race. Proves the lock yields consistent reads, not just
// absence of a detector warning.
func TestAdv_APIKeyStore_ConcurrentRevokeThenValidateConsistent(t *testing.T) {
	store := NewAPIKeyStore()
	store.AddKey("k", APIKeyInfo{Key: "k", UserID: "u", Active: true})

	var wg sync.WaitGroup
	wg.Add(2)
	// Revoker
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			store.RevokeKey("k")
		}
	}()
	// Validator — once we observe a deny it must stay denied (revocation is
	// monotonic here: nothing re-activates "k").
	go func() {
		defer wg.Done()
		sawDeny := false
		for i := 0; i < 1000; i++ {
			_, ok := store.ValidateKey("k")
			if sawDeny && ok {
				t.Errorf("SECURITY: key re-validated after a revoke was observed (inconsistent read)")
				return
			}
			if !ok {
				sawDeny = true
			}
		}
	}()
	wg.Wait()
}
