package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"mikhailjbs/chat-service/internal/infra/security"
)

const (
	DefaultAccessTokenCookie = "access_token"
	DefaultClaimsContextKey  = "auth.claims"
)

type Config struct {
	TokenManager      *security.TokenManager
	AccessTokenCookie string
	ContextKey        string
	AllowQueryToken   bool
}

type Policy struct {
	Roles          []string
	AllowAnonymous bool
}

type AuthMiddleware struct {
	tokenManager *security.TokenManager
	accessCookie string
	contextKey   string
	allowQuery   bool
}

func NewAuthMiddleware(cfg Config) *AuthMiddleware {
	if cfg.TokenManager == nil {
		panic("middleware.NewAuthMiddleware: TokenManager is required")
	}
	if cfg.AccessTokenCookie == "" {
		cfg.AccessTokenCookie = DefaultAccessTokenCookie
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = DefaultClaimsContextKey
	}

	return &AuthMiddleware{
		tokenManager: cfg.TokenManager,
		accessCookie: cfg.AccessTokenCookie,
		contextKey:   cfg.ContextKey,
		allowQuery:   cfg.AllowQueryToken,
	}
}

func (a *AuthMiddleware) Require(policy Policy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := a.extractToken(c)
		if token == "" {
			if policy.AllowAnonymous {
				return c.Next()
			}
			return unauthorized(c, "missing access token")
		}

		claims, err := a.tokenManager.ParseAccessToken(token)
		if err != nil {
			if policy.AllowAnonymous {
				return c.Next()
			}
			return unauthorized(c, "invalid access token")
		}

		if len(policy.Roles) > 0 && !hasIntersection(claims.Roles, policy.Roles) {
			return forbidden(c, "insufficient permissions")
		}

		c.Locals(a.contextKey, claims)
		return c.Next()
	}
}

func ClaimsFromContext(c *fiber.Ctx) (*security.ClaimsPayload, bool) {
	return ClaimsFromContextWithKey(c, DefaultClaimsContextKey)
}

func ClaimsFromContextWithKey(c *fiber.Ctx, key string) (*security.ClaimsPayload, bool) {
	val := c.Locals(key)
	if val == nil {
		return nil, false
	}
	if claims, ok := val.(*security.ClaimsPayload); ok {
		return claims, true
	}
	if claims, ok := val.(security.ClaimsPayload); ok {
		return &claims, true
	}
	return nil, false
}

func (a *AuthMiddleware) extractToken(c *fiber.Ctx) string {
	if token := extractBearerToken(c.Get("Authorization")); token != "" {
		return token
	}
	if token := c.Cookies(a.accessCookie); token != "" {
		return token
	}
	if a.allowQuery {
		if token := c.Query("token"); token != "" {
			return token
		}
	}
	return ""
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	header = strings.TrimSpace(header)
	const prefix = "Bearer "
	if strings.HasPrefix(strings.ToLower(header), strings.ToLower(prefix)) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return header
}

func hasIntersection(userRoles, required []string) bool {
	roleSet := make(map[string]struct{}, len(userRoles))
	for _, r := range userRoles {
		roleSet[strings.ToLower(r)] = struct{}{}
	}
	for _, req := range required {
		if _, ok := roleSet[strings.ToLower(req)]; ok {
			return true
		}
	}
	return false
}

func unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(errorResponse{
		Ok:     false,
		Status: fiber.StatusUnauthorized,
		Error:  message,
	})
}

func forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(errorResponse{
		Ok:     false,
		Status: fiber.StatusForbidden,
		Error:  message,
	})
}

type errorResponse struct {
	Ok     bool   `json:"ok"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}
