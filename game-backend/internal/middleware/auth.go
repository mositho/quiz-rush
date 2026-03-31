package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"quiz-rush/game-backend/internal/httpjson"

	"github.com/coreos/go-oidc/v3/oidc"
)

type authenticatedUserContextKey struct{}

type AuthenticatedUser struct {
	Subject           string
	PreferredUsername string
	Email             string
	AuthorizedParty   string
	Audience          []string
}

type accessTokenClaims struct {
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	AuthorizedParty   string `json:"azp"`
}

func NewOIDCAuthMiddleware(
	ctx context.Context,
	issuerURL string,
	internalIssuerURL string,
	clientID string,
) (func(http.Handler) http.Handler, error) {
	verifier, err := newOIDCVerifier(ctx, issuerURL, internalIssuerURL, clientID)
	if err != nil {
		return nil, err
	}
	if verifier == nil {
		return nil, nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authState, err := authenticateRequest(verifier, clientID, r)
			if err != nil {
				log.Printf("auth rejected: %v", err)
				writeUnauthorized(w)
				return
			}
			if authState == authStateMissing {
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authenticatedUserContextKey{}, user)))
		})
	}, nil
}

func AuthenticatedUserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey{}).(AuthenticatedUser)
	return user, ok
}

func WithAuthenticatedUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey{}, user)
}

func containsAudience(audience []string, clientID string) bool {
	for _, entry := range audience {
		if entry == clientID {
			return true
		}
	}

	return false
}

func writeUnauthorized(w http.ResponseWriter) {
	if err := httpjson.Write(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}); err != nil {
		log.Printf("failed to write unauthorized response: %v", err)
	}
}

type requestAuthState int

const (
	authStateMissing requestAuthState = iota
	authStatePresent
)

func newOIDCVerifier(
	ctx context.Context,
	issuerURL string,
	internalIssuerURL string,
	clientID string,
) (*oidc.IDTokenVerifier, error) {
	if issuerURL == "" || internalIssuerURL == "" || clientID == "" {
		return nil, nil
	}

	_ = ctx
	jwksURL := strings.TrimSuffix(internalIssuerURL, "/") + "/protocol/openid-connect/certs"
	keySet := oidc.NewRemoteKeySet(context.Background(), jwksURL)
	return oidc.NewVerifier(issuerURL, keySet, &oidc.Config{SkipClientIDCheck: true}), nil
}

func authenticateRequest(verifier *oidc.IDTokenVerifier, clientID string, r *http.Request) (AuthenticatedUser, requestAuthState, error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		return AuthenticatedUser{}, authStateMissing, nil
	}
	if !strings.HasPrefix(authorization, "Bearer ") {
		return AuthenticatedUser{}, authStatePresent, fmt.Errorf("missing bearer token")
	}

	rawToken := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if rawToken == "" {
		return AuthenticatedUser{}, authStatePresent, fmt.Errorf("empty bearer token")
	}

	verifiedToken, err := verifier.Verify(r.Context(), rawToken)
	if err != nil {
		return AuthenticatedUser{}, authStatePresent, fmt.Errorf("token verification failed: %w", err)
	}

	var claims accessTokenClaims
	if err := verifiedToken.Claims(&claims); err != nil {
		return AuthenticatedUser{}, authStatePresent, fmt.Errorf("unable to decode token claims: %w", err)
	}

	if claims.AuthorizedParty != clientID && !containsAudience(verifiedToken.Audience, clientID) {
		return AuthenticatedUser{}, authStatePresent, fmt.Errorf(
			"token client mismatch, azp=%q aud=%q expected=%q",
			claims.AuthorizedParty,
			verifiedToken.Audience,
			clientID,
		)
	}

	return AuthenticatedUser{
		Subject:           claims.Subject,
		PreferredUsername: claims.PreferredUsername,
		Email:             claims.Email,
		AuthorizedParty:   claims.AuthorizedParty,
		Audience:          append([]string(nil), verifiedToken.Audience...),
	}, authStatePresent, nil
}
