package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	providerContext := ctx
	if issuerURL != internalIssuerURL {
		providerContext = oidc.InsecureIssuerURLContext(providerContext, issuerURL)
	}

	provider, err := newProviderWithRetry(providerContext, internalIssuerURL, 150, 2*time.Second)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				writeUnauthorized(w)
				return
			}

			rawToken := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
			if rawToken == "" {
				writeUnauthorized(w)
				return
			}

			verifiedToken, err := verifier.Verify(r.Context(), rawToken)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			var claims accessTokenClaims
			if err := verifiedToken.Claims(&claims); err != nil {
				writeUnauthorized(w)
				return
			}

			if claims.AuthorizedParty != clientID {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

func newProviderWithRetry(
	ctx context.Context,
	internalIssuerURL string,
	attempts int,
	wait time.Duration,
) (*oidc.Provider, error) {
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		provider, err := oidc.NewProvider(ctx, internalIssuerURL)
		if err == nil {
			return provider, nil
		}

		lastErr = err

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("oidc provider unavailable after %d attempts: %w", attempts, lastErr)
}

func writeUnauthorized(w http.ResponseWriter) {
	httpjson.Write(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
