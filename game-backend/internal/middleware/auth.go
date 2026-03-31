package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"quiz-rush/game-backend/internal/httpjson"

	"github.com/coreos/go-oidc/v3/oidc"
)

type accessTokenClaims struct {
	AuthorizedParty string `json:"azp"`
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

			next.ServeHTTP(w, r)
		})
	}, nil
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
