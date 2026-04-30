package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"mikhailjbs/chat-service/internal/infra/security"
)

const (
	DefaultClaimsContextKey = "auth.claims"
)

type Config struct {
	ContextKey string
}

type Policy struct {
	AllowAnonymous bool
	Roles          []string
}

type AuthMiddleware struct {
	contextKey string
}

func NewAuthMiddleware(cfg Config) *AuthMiddleware {
	if cfg.ContextKey == "" {
		cfg.ContextKey = DefaultClaimsContextKey
	}
	return &AuthMiddleware{
		contextKey: cfg.ContextKey,
	}
}

func (a *AuthMiddleware) Require(policy Policy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Get("X-User-Id")
		if userID == "" {
			userID = c.Get("X-User-ID")
		}

		if userID == "" {
			if policy.AllowAnonymous {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		rolesStr := c.Get("X-User-Roles")
		roles := []string{}
		if rolesStr != "" {
			roles = strings.Split(rolesStr, ",")
		}

		claims := &security.ClaimsPayload{
			UserID:   userID,
			Email:    c.Get("X-User-Email"),
			Username: c.Get("X-User-Username"),
			Roles:    roles,
		}

		if len(policy.Roles) > 0 && !hasIntersection(claims.Roles, policy.Roles) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
		}

		c.Locals(a.contextKey, claims)
		return c.Next()
	}
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

func ClaimsFromContext(c *fiber.Ctx) (*security.ClaimsPayload, bool) {
	val := c.Locals(DefaultClaimsContextKey)
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
