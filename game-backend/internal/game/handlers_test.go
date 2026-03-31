package game

import "testing"

func TestAuthorizeSessionAccessAllowsAnonymousSession(t *testing.T) {
	if err := authorizeSessionAccess(nil, nil, true); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizeSessionAccessRequiresAuthenticationForOwnedSession(t *testing.T) {
	ownerProfileID := "owner-profile"

	err := authorizeSessionAccess(nil, &ownerProfileID, false)
	if err != errAuthenticationRequired {
		t.Fatalf("got %v, want %v", err, errAuthenticationRequired)
	}
}

func TestAuthorizeSessionAccessRejectsDifferentAuthenticatedOwner(t *testing.T) {
	authenticatedProfileID := "profile-a"
	ownerProfileID := "profile-b"

	err := authorizeSessionAccess(&authenticatedProfileID, &ownerProfileID, false)
	if err != errSessionForbidden {
		t.Fatalf("got %v, want %v", err, errSessionForbidden)
	}
}

func TestAuthorizeSessionAccessAllowsMatchingAuthenticatedOwner(t *testing.T) {
	authenticatedProfileID := "profile-a"
	ownerProfileID := "profile-a"

	if err := authorizeSessionAccess(&authenticatedProfileID, &ownerProfileID, false); err != nil {
		t.Fatal(err)
	}
}
