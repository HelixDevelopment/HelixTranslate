package security

// Adversarial / attack-shaped security tests (§11.4.27 SECURITY test type).
//
// These are deliberately distinct from the happy-path unit tests already in
// this package. Every assertion below reflects a REAL attack and FAILS if the
// security check it exercises were bypassed or stubbed to always-accept
// (anti-bluff per §11.4 / §11.4.1). All test functions use the TestAdv_ prefix
// to avoid any collision with the existing _test.go files in this package.
//
// CORS NOTE: pkg/security/ ships no CORS implementation (CLAUDE.md lists CORS
// under pkg/security but no Access-Control / CORS source exists here at this
// revision). CORS adversarial coverage is therefore reported as UNCONFIRMED /
// not-present rather than asserted against a non-existent surface — asserting
// against absent code would be a §11.4.1 FAIL-bluff.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const advSecret = "adversarial-test-secret-key-32bytes!" // >= 16 chars (NewAuthService requirement)

func advNewAuth(t *testing.T) *AuthService {
	t.Helper()
	return NewAuthService(advSecret, time.Hour)
}

// ---------------------------------------------------------------------------
// JWT adversarial tests
// ---------------------------------------------------------------------------

// TestAdv_JWT_TamperedSignatureRejected — flipping any byte of the signature
// segment must be rejected. Attack: token forgery by signature mutation.
func TestAdv_JWT_TamperedSignatureRejected(t *testing.T) {
	as := advNewAuth(t)
	tok, err := as.GenerateToken("u1", "alice", []string{"user"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}
	// Mutate the signature: flip the last character to a different valid base64url char.
	sig := []byte(parts[2])
	if sig[len(sig)-1] == 'A' {
		sig[len(sig)-1] = 'B'
	} else {
		sig[len(sig)-1] = 'A'
	}
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	if _, err := as.ValidateToken(tampered); err == nil {
		t.Fatal("SECURITY: tampered-signature token was ACCEPTED — forgery possible")
	}
}

// TestAdv_JWT_TamperedPayloadRejected — modifying the claims (privilege
// escalation: user -> admin) without re-signing must be rejected.
func TestAdv_JWT_TamperedPayloadRejected(t *testing.T) {
	as := advNewAuth(t)
	tok, err := as.GenerateToken("u1", "alice", []string{"user"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	parts := strings.Split(tok, ".")

	// Decode payload, escalate roles to admin, re-encode WITHOUT re-signing.
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	m["roles"] = []string{"admin"}
	mutated, _ := json.Marshal(m)
	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString(mutated) + "." + parts[2]

	if _, err := as.ValidateToken(forged); err == nil {
		t.Fatal("SECURITY: payload-tampered (role-escalated) token was ACCEPTED")
	}
}

// TestAdv_JWT_AlgNoneRejected — the classic alg=none attack: attacker crafts a
// header {"alg":"none"} and empty signature. MUST be rejected.
func TestAdv_JWT_AlgNoneRejected(t *testing.T) {
	as := advNewAuth(t)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"user_id":"attacker","username":"attacker","roles":["admin"],"exp":99999999999}`))

	// Several "none" shapes attackers try.
	for _, tok := range []string{
		header + "." + payload + ".",   // empty signature
		header + "." + payload,         // no signature segment
		header + "." + payload + ".AA", // junk signature
	} {
		if _, err := as.ValidateToken(tok); err == nil {
			t.Fatalf("SECURITY: alg=none token was ACCEPTED: %q", tok)
		}
	}
}

// TestAdv_JWT_AlgConfusionRSToHSRejected — algorithm-confusion attack. An
// attacker takes an RS256-style token (asymmetric) and tries to get it
// validated as HS256 using the public material as the HMAC key. The verifier
// only accepts *SigningMethodHMAC, so an RS256-header token must be rejected.
func TestAdv_JWT_AlgConfusionRSToHSRejected(t *testing.T) {
	as := advNewAuth(t)

	// Forge a token whose header declares RS256 but is HMAC-signed with the
	// secret (the shape of an alg-confusion forgery attempt against a naive
	// verifier that keys HMAC off whatever the header says).
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"user_id":"attacker","username":"attacker","roles":["admin"]}`))
	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, []byte(advSecret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	forged := signingInput + "." + sig

	if _, err := as.ValidateToken(forged); err == nil {
		t.Fatal("SECURITY: RS256-header (alg-confusion) token was ACCEPTED as HMAC")
	}
}

// TestAdv_JWT_ExpiredRejected — an expired token must be rejected.
func TestAdv_JWT_ExpiredRejected(t *testing.T) {
	as := advNewAuth(t)

	claims := Claims{
		UserID:   "u1",
		Username: "alice",
		Roles:    []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // already expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(advSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := as.ValidateToken(tok); err == nil {
		t.Fatal("SECURITY: expired token was ACCEPTED")
	}
}

// TestAdv_JWT_FutureNbfRejected — a token whose not-before is in the future
// must be rejected (token-not-yet-valid).
func TestAdv_JWT_FutureNbfRejected(t *testing.T) {
	as := advNewAuth(t)

	claims := Claims{
		UserID:   "u1",
		Username: "alice",
		Roles:    []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // future
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(advSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := as.ValidateToken(tok); err == nil {
		t.Fatal("SECURITY: token with future nbf was ACCEPTED")
	}
}

// TestAdv_JWT_WrongSecretRejected — a token signed with a different secret must
// be rejected. Attack: forgery without the real signing key.
func TestAdv_JWT_WrongSecretRejected(t *testing.T) {
	as := advNewAuth(t)

	claims := Claims{
		UserID:   "u1",
		Username: "alice",
		Roles:    []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte("a-totally-different-attacker-secret-key"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := as.ValidateToken(tok); err == nil {
		t.Fatal("SECURITY: token signed with wrong secret was ACCEPTED")
	}
}

// TestAdv_JWT_MalformedAndEmptyRejected — garbage / empty / structurally-broken
// tokens must all be rejected, never panic.
func TestAdv_JWT_MalformedAndEmptyRejected(t *testing.T) {
	as := advNewAuth(t)

	for _, tok := range []string{
		"",
		"   ",
		"not-a-jwt",
		"only.two",
		"a.b.c",
		"....",
		strings.Repeat("A", 4096), // oversized junk
		"eyJ.eyJ.",
	} {
		if _, err := as.ValidateToken(tok); err == nil {
			t.Fatalf("SECURITY: malformed/empty token was ACCEPTED: %q", tok)
		}
	}
}

// TestAdv_JWT_ValidTokenAcceptedWithClaims — positive control. Proves the
// negative tests above fail for a real reason (rejection), not because the
// verifier rejects everything. A genuine token MUST be accepted and round-trip
// the exact claims. If this ever fails together with the negatives passing, the
// verifier is reject-all (also a bluff).
func TestAdv_JWT_ValidTokenAcceptedWithClaims(t *testing.T) {
	as := advNewAuth(t)
	tok, err := as.GenerateToken("u-42", "alice", []string{"user", "editor"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := as.ValidateToken(tok)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if claims.UserID != "u-42" || claims.Username != "alice" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "user" || claims.Roles[1] != "editor" {
		t.Fatalf("roles not round-tripped: %v", claims.Roles)
	}
}

// ---------------------------------------------------------------------------
// Rate-limiter adversarial tests
// ---------------------------------------------------------------------------

// TestAdv_RateLimiter_BurstBeyondLimitThrottled — drive burst+overflow requests
// for one key; the (burst+1)th MUST be rejected. Attack: request flooding.
func TestAdv_RateLimiter_BurstBeyondLimitThrottled(t *testing.T) {
	const burst = 5
	rl := NewRateLimiter(1, burst) // 1 rps, burst 5

	allowed := 0
	for i := 0; i < burst; i++ {
		if rl.Allow("attacker") {
			allowed++
		}
	}
	if allowed != burst {
		t.Fatalf("expected first %d requests allowed, got %d", burst, allowed)
	}
	// The (burst+1)th request within the same instant MUST be throttled.
	if rl.Allow("attacker") {
		t.Fatal("SECURITY: request beyond burst limit was ALLOWED — flood not throttled")
	}
}

// TestAdv_RateLimiter_WindowResetReplenishes — after the refill window elapses,
// at least one new token must be granted (limiter not permanently jammed) but
// only after waiting, not instantly. Confirms throttle is time-bounded, not
// fake-permanent.
func TestAdv_RateLimiter_WindowResetReplenishes(t *testing.T) {
	rl := NewRateLimiter(20, 1) // 20 rps -> refill ~every 50ms, burst 1

	if !rl.Allow("k") {
		t.Fatal("first request should be allowed")
	}
	if rl.Allow("k") {
		t.Fatal("SECURITY: second immediate request should be throttled (burst=1)")
	}
	// Wait longer than one refill interval (1/20s = 50ms).
	time.Sleep(120 * time.Millisecond)
	if !rl.Allow("k") {
		t.Fatal("after window elapsed, a replenished token should be granted")
	}
}

// TestAdv_RateLimiter_PerKeyIsolation — one key exhausting its budget MUST NOT
// throttle a different key. Attack: a noisy/malicious tenant degrading others.
func TestAdv_RateLimiter_PerKeyIsolation(t *testing.T) {
	const burst = 3
	rl := NewRateLimiter(1, burst)

	// Exhaust "victim-neighbor"... no, exhaust the attacker key fully.
	for i := 0; i < burst; i++ {
		rl.Allow("attacker")
	}
	if rl.Allow("attacker") {
		t.Fatal("attacker key should be exhausted")
	}
	// A different key must still have its full fresh budget.
	for i := 0; i < burst; i++ {
		if !rl.Allow("honest-user") {
			t.Fatalf("SECURITY: honest-user blocked by attacker's exhaustion at req %d", i)
		}
	}
}

// TestAdv_RateLimiter_ConcurrentContention — drive many goroutines at one key
// and assert the number of ALLOWED requests never exceeds burst within the
// instant (no over-admission under races). Run this with -race.
func TestAdv_RateLimiter_ConcurrentContention(t *testing.T) {
	const burst = 10
	const goroutines = 200
	rl := NewRateLimiter(1, burst) // slow refill so the cap is dominated by burst

	var allowed int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if rl.Allow("shared-key") {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	// With rps=1 and burst=10, across a sub-second window we may get the 10
	// burst tokens plus possibly 1 refill token. Allow a small slack but assert
	// it is NOT unbounded (the attack we guard against: over-admission).
	if got := atomic.LoadInt64(&allowed); got > burst+2 {
		t.Fatalf("SECURITY: over-admitted under contention: allowed=%d (burst=%d)", got, burst)
	}
	if got := atomic.LoadInt64(&allowed); got < 1 {
		t.Fatalf("rate limiter admitted nothing (got=%d) — broken, not just strict", got)
	}
}

// TestAdv_RateLimiter_ResetClearsState — Reset must drop a key's limiter so a
// previously-exhausted key gets a fresh budget (admin recovery path), and must
// not leak across keys.
func TestAdv_RateLimiter_ResetClearsState(t *testing.T) {
	const burst = 2
	rl := NewRateLimiter(1, burst)

	for i := 0; i < burst; i++ {
		rl.Allow("k")
	}
	if rl.Allow("k") {
		t.Fatal("key should be exhausted before reset")
	}
	rl.Reset("k")
	if !rl.Allow("k") {
		t.Fatal("after Reset the key should have a fresh budget")
	}
}
