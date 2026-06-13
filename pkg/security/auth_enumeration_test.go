package security

// Account-status / username-enumeration oracle adversarial tests
// (§11.4.27 SECURITY test type, §11.4.115 reproduce-first).
//
// ROOT CAUSE (FACT): UserAuthService.AuthenticateUser checked user.IsActive
// BEFORE validating the password. For an existing-but-inactive account, an
// unauthenticated attacker submitting ANY (wrong) password received
// models.ErrUserInactive, whereas a wrong password on an active account — or a
// non-existent username — returned models.ErrInvalidCredentials. The production
// handler (pkg/api/handler.go login) maps ErrUserInactive -> HTTP 403 and
// ErrInvalidCredentials -> HTTP 401, so the discrepancy is observable over the
// network: the attacker learns "this username exists and is inactive" without
// ever knowing the password. CWE-204 (observable response discrepancy) /
// CWE-203 (information exposure through discrepancy) — username enumeration.
//
// SECURE BEHAVIOUR: the password MUST be validated first; an attacker who does
// not know the password MUST get the SAME response for an inactive account as
// for a wrong password / unknown user (ErrInvalidCredentials). Only a caller
// presenting the CORRECT password may be told the account is inactive.
//
// These tests FAIL on the pre-fix code (oracle present) and PASS after the fix.

import (
	"testing"
	"time"

	"digital.vasic.translator/pkg/models"
)

// newEnumTestUser builds a user with a real bcrypt-hashed password so
// ValidatePassword behaves exactly as in production.
func newEnumTestUser(t *testing.T, id, username string, active bool, plaintext string) *models.User {
	t.Helper()
	u := &models.User{
		ID:       id,
		Username: username,
		Email:    username + "@example.com",
		IsActive: active,
		Roles:    []string{"user"},
	}
	if err := u.SetPassword(plaintext); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	return u
}

// TestAdv_AuthenticateUser_InactiveAccountNoEnumerationOracle is the core RED
// test. An attacker who does NOT know the password must not be able to tell an
// inactive account apart from a wrong-password / unknown-user response.
func TestAdv_AuthenticateUser_InactiveAccountNoEnumerationOracle(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["activeuser"] = newEnumTestUser(t, "id-active", "activeuser", true, "correct-horse")
	repo.users["inactiveuser"] = newEnumTestUser(t, "id-inactive", "inactiveuser", false, "correct-horse")

	uas := NewUserAuthService("enumeration-test-secret-key-32!", time.Hour, repo)

	// Baseline: wrong password on an ACTIVE account -> ErrInvalidCredentials.
	_, errActiveWrong := uas.AuthenticateUser(LoginRequest{Username: "activeuser", Password: "wrong"})
	if errActiveWrong != models.ErrInvalidCredentials {
		t.Fatalf("active+wrong-password should be ErrInvalidCredentials, got %v", errActiveWrong)
	}

	// Baseline: unknown username -> ErrInvalidCredentials.
	_, errUnknown := uas.AuthenticateUser(LoginRequest{Username: "ghost", Password: "wrong"})
	if errUnknown != models.ErrInvalidCredentials {
		t.Fatalf("unknown-user should be ErrInvalidCredentials, got %v", errUnknown)
	}

	// ATTACK: wrong password on an INACTIVE account. A secure system reveals
	// nothing more than the wrong-password baseline above. The pre-fix code
	// leaked ErrUserInactive here, distinguishing the account => enumeration.
	_, errInactiveWrong := uas.AuthenticateUser(LoginRequest{Username: "inactiveuser", Password: "wrong"})
	if errInactiveWrong != models.ErrInvalidCredentials {
		t.Fatalf("SECURITY: account-status ENUMERATION ORACLE — wrong password on an "+
			"inactive account returned %v but the wrong-password baseline is %v. An "+
			"unauthenticated attacker can distinguish existing inactive accounts. "+
			"Validate the password BEFORE reporting IsActive.",
			errInactiveWrong, models.ErrInvalidCredentials)
	}
}

// TestAdv_AuthenticateUser_InactiveRevealedOnlyWithCorrectPassword is the
// positive control: a caller presenting the CORRECT password for an inactive
// account is legitimately told the account is inactive (so the fix does not
// over-correct into hiding inactivity from the real owner).
func TestAdv_AuthenticateUser_InactiveRevealedOnlyWithCorrectPassword(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["inactiveuser"] = newEnumTestUser(t, "id-inactive", "inactiveuser", false, "correct-horse")

	uas := NewUserAuthService("enumeration-test-secret-key-32!", time.Hour, repo)

	_, err := uas.AuthenticateUser(LoginRequest{Username: "inactiveuser", Password: "correct-horse"})
	if err != models.ErrUserInactive {
		t.Fatalf("correct password on inactive account should report ErrUserInactive, got %v", err)
	}
}

// TestAdv_AuthenticateUser_ActiveCorrectPasswordStillSucceeds is the happy-path
// control proving the reordered checks did not break normal login.
func TestAdv_AuthenticateUser_ActiveCorrectPasswordStillSucceeds(t *testing.T) {
	repo := NewMockUserRepository()
	repo.users["activeuser"] = newEnumTestUser(t, "id-active", "activeuser", true, "correct-horse")

	uas := NewUserAuthService("enumeration-test-secret-key-32!", time.Hour, repo)

	resp, err := uas.AuthenticateUser(LoginRequest{Username: "activeuser", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("active user with correct password should log in, got %v", err)
	}
	if resp == nil || resp.Token == "" || resp.UserID != "id-active" {
		t.Fatalf("login response malformed: %+v", resp)
	}
}
