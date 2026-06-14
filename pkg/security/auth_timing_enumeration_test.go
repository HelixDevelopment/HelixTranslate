package security

// Username-enumeration via login-TIMING oracle adversarial test
// (§11.4.27 SECURITY test type, §11.4.115 reproduce-first, §11.4.146 STEP 1-3).
//
// ROOT CAUSE (FACT): UserAuthService.AuthenticateUser short-circuits the moment
// FindByUsername reports ErrUserNotFound — it returns ErrInvalidCredentials
// WITHOUT performing any bcrypt comparison. For an EXISTING user with a wrong
// password it instead runs the full bcrypt.CompareHashAndPassword (DefaultCost,
// ~tens of milliseconds). The wall-clock cost of the two paths therefore differs
// by orders of magnitude, so an unauthenticated attacker who times the login
// response learns whether a username exists — even though both paths return the
// identical ErrInvalidCredentials error value. This is CWE-208 (observable
// timing discrepancy) / CWE-204 username enumeration, DISTINCT from the
// already-fixed active-vs-inactive error-VALUE oracle: the error values now
// match, but the response TIME does not.
//
// SECURE BEHAVIOUR: the unknown-user path MUST perform a comparable bcrypt
// computation (a "dummy hash" compare) so the response time does not reveal user
// existence. This is the standard mitigation (Django check_password against an
// unusable hash, OWASP "Authentication and Error Messages" guidance).
//
// This test FAILS on the pre-fix code (unknown-user path is dramatically faster)
// and PASSES after the fix (both paths pay the bcrypt cost).

import (
	"testing"
	"time"

	"digital.vasic.translator/pkg/models"
)

// medianAuthDuration runs AuthenticateUser n times and returns the median wall
// clock duration. Median (not mean) is used to suppress GC / scheduler spikes.
func medianAuthDuration(t *testing.T, uas *UserAuthService, req LoginRequest, n int) time.Duration {
	t.Helper()
	ds := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		_, _ = uas.AuthenticateUser(req)
		ds = append(ds, time.Since(start))
	}
	// simple insertion sort (n is small) then pick the middle element.
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
			ds[j], ds[j-1] = ds[j-1], ds[j]
		}
	}
	return ds[len(ds)/2]
}

// TestAdv_AuthenticateUser_NoUsernameTimingOracle is the core RED test. The
// unknown-user login path must not be dramatically faster than the
// known-user-wrong-password path; if it is, response time leaks user existence.
func TestAdv_AuthenticateUser_NoUsernameTimingOracle(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["realuser"] = newEnumTestUser(t, "id-real", "realuser", true, "correct-horse-battery")

	uas := NewUserAuthService("timing-enum-test-secret-key-32!!", time.Hour, repo)

	const iters = 9

	// Known user, WRONG password -> always runs bcrypt over the stored hash.
	knownWrong := medianAuthDuration(t, uas,
		LoginRequest{Username: "realuser", Password: "definitely-wrong"}, iters)

	// Unknown user -> on vulnerable code returns before any bcrypt work.
	unknown := medianAuthDuration(t, uas,
		LoginRequest{Username: "ghost-does-not-exist", Password: "definitely-wrong"}, iters)

	// On the SECURE implementation both paths pay a bcrypt compare, so their
	// medians are the same order of magnitude. On the VULNERABLE implementation
	// the unknown-user path skips bcrypt and is typically 100x-1000x faster.
	//
	// Anti-bluff calibration: a real bcrypt compare at DefaultCost is well above
	// 1ms on any realistic host; a map-miss return is sub-microsecond. We assert
	// the known-wrong path is NOT more than 10x the unknown path. A 10x ratio is
	// far below the real >100x gap of the vulnerable code (so no false RED from
	// scheduler noise) yet far above the ~1x gap of the fixed code.
	if knownWrong > 10*unknown {
		t.Fatalf("SECURITY: username-enumeration TIMING ORACLE — known-user-wrong-password "+
			"median=%v but unknown-user median=%v (%.1fx faster for unknown). The "+
			"unknown-user path skips the bcrypt compare, so response time reveals whether "+
			"a username exists. Perform a dummy bcrypt compare on the user-not-found path.",
			knownWrong, unknown, float64(knownWrong)/float64(unknown+1))
	}
}

// TestAdv_AuthenticateUser_UnknownUserStillRejected is the functional control
// (§11.4.146 STEP 3): the timing mitigation must not change the security verdict
// — an unknown user is still rejected with ErrInvalidCredentials, and a known
// user with the correct password still logs in.
func TestAdv_AuthenticateUser_UnknownUserStillRejected(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["realuser"] = newEnumTestUser(t, "id-real", "realuser", true, "correct-horse-battery")

	uas := NewUserAuthService("timing-enum-test-secret-key-32!!", time.Hour, repo)

	if _, err := uas.AuthenticateUser(
		LoginRequest{Username: "ghost", Password: "whatever"}); err != models.ErrInvalidCredentials {
		t.Fatalf("unknown user must be ErrInvalidCredentials, got %v", err)
	}
	resp, err := uas.AuthenticateUser(
		LoginRequest{Username: "realuser", Password: "correct-horse-battery"})
	if err != nil || resp == nil || resp.Token == "" {
		t.Fatalf("known user + correct password must log in, got err=%v resp=%+v", err, resp)
	}
}
