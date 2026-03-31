package middleware

import (
	"context"
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
	if issuerURL == "" || internalIssuerURL == "" || clientID == "" {
		return nil, nil
	}

	_ = ctx
	jwksURL := strings.TrimSuffix(internalIssuerURL, "/") + "/protocol/openid-connect/certs"
	keySet := oidc.NewRemoteKeySet(context.Background(), jwksURL)
	verifier := oidc.NewVerifier(issuerURL, keySet, &oidc.Config{SkipClientIDCheck: true})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				log.Printf("auth rejected: missing bearer token")
				writeUnauthorized(w)
				return
			}

			rawToken := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
			if rawToken == "" {
				log.Printf("auth rejected: empty bearer token")
				writeUnauthorized(w)
				return
			}

			verifiedToken, err := verifier.Verify(r.Context(), rawToken)
			if err != nil {
				log.Printf("auth rejected: token verification failed: %v", err)
				writeUnauthorized(w)
				return
			}

			var claims accessTokenClaims
			if err := verifiedToken.Claims(&claims); err != nil {
				log.Printf("auth rejected: unable to decode token claims: %v", err)
				writeUnauthorized(w)
				return
			}

			if claims.AuthorizedParty != clientID && !containsAudience(verifiedToken.Audience, clientID) {
				log.Printf(
					"auth rejected: token client mismatch, azp=%q aud=%q expected=%q",
					claims.AuthorizedParty,
					verifiedToken.Audience,
					clientID,
				)
				writeUnauthorized(w)
				return
			}

			user := AuthenticatedUser{
				Subject:           claims.Subject,
				PreferredUsername: claims.PreferredUsername,
				Email:             claims.Email,
				AuthorizedParty:   claims.AuthorizedParty,
				Audience:          append([]string(nil), verifiedToken.Audience...),
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
	httpjson.Write(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
