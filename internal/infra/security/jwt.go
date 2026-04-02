package security

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TokenManager provides methods to create and verify access & refresh tokens.
type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	logger        *logrus.Logger
}

// TokenPair is the pair of tokens returned on login/refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessExp    time.Time
	RefreshExp   time.Time
	JTI          string // unique id for the access token
	SID          string // session id associated with refresh token
}

// ClaimsPayload holds parsed token data you care about.
type ClaimsPayload struct {
	UserID   string
	Email    string
	Username string
	Roles    []string
	JTI      string
	SID      string
	Expiry   time.Time
}

// NewTokenManager creates a TokenManager. Both secrets must be non-empty.
// accessTTL and refreshTTL define token lifetimes.
func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, logger *logrus.Logger) (*TokenManager, error) {
	if accessSecret == "" || refreshSecret == "" {
		return nil, errors.New("access and refresh secrets must be provided")
	}
	if logger == nil {
		// if not provided, create a minimal logger
		logger = logrus.New()
	}
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		logger:        logger,
	}, nil
}

// GenerateTokenPair issues an access + refresh token for the given subject details.
// userID: string identifier (uuid). roles: list of roles (eg "user","admin").
// Returns TokenPair where JTI is access token id and SID is session id (refresh).
func (t *TokenManager) GenerateTokenPair(userID, email, username string, roles []string) (*TokenPair, error) {
	now := time.Now().UTC()
	accessExp := now.Add(t.accessTTL)
	refreshExp := now.Add(t.refreshTTL)

	jti := uuid.NewString() // unique id for access token
	sid := uuid.NewString() // session id for refresh token

	// Access token claims
	accessClaims := jwt.MapClaims{
		"sub":        userID,
		"email":      email,
		"username":   username,
		"roles":      roles,
		"jti":        jti,
		"session_id": sid,
		"iat":        now.Unix(),
		"exp":        accessExp.Unix(),
		"nbf":        now.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(t.accessSecret)
	if err != nil {
		t.logger.WithError(err).Error("failed to sign access token")
		return nil, err
	}

	// Refresh token claims - keep minimal, but include sid so we can tie it to session
	refreshClaims := jwt.MapClaims{
		"sub":        userID,
		"session_id": sid,
		"jti":        uuid.NewString(), // refresh token has its own jti if needed
		"iat":        now.Unix(),
		"exp":        refreshExp.Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(t.refreshSecret)
	if err != nil {
		t.logger.WithError(err).Error("failed to sign refresh token")
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		JTI:          jti,
		SID:          sid,
	}, nil
}

// ParseAccessToken parses and validates an access token and returns ClaimsPayload.
func (t *TokenManager) ParseAccessToken(tokenStr string) (*ClaimsPayload, error) {
	if tokenStr == "" {
		return nil, errors.New("token empty")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// only accept HMAC signing
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return t.accessSecret, nil
	}, jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claimsToPayload(claims)
}

// ParseRefreshToken parses and validates a refresh token.
func (t *TokenManager) ParseRefreshToken(tokenStr string) (*ClaimsPayload, error) {
	if tokenStr == "" {
		return nil, errors.New("token empty")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return t.refreshSecret, nil
	}, jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claimsToPayload(claims)
}

func (t *TokenManager) AccessTTL() time.Duration {
	return t.accessTTL
}

func (t *TokenManager) RefreshTTL() time.Duration {
	return t.refreshTTL
}

// claimsToPayload converts MapClaims into our ClaimsPayload structure.
func claimsToPayload(claims jwt.MapClaims) (*ClaimsPayload, error) {
	var cp ClaimsPayload

	if sub, ok := claims["sub"]; ok {
		switch v := sub.(type) {
		case string:
			cp.UserID = v
		case float64:
			cp.UserID = strconv.FormatInt(int64(v), 10)
		case int64:
			cp.UserID = strconv.FormatInt(v, 10)
		case int:
			cp.UserID = strconv.FormatInt(int64(v), 10)
		}
	}

	if email, ok := claims["email"].(string); ok {
		cp.Email = email
	}
	if un, ok := claims["username"].(string); ok {
		cp.Username = un
	}
	if jti, ok := claims["jti"].(string); ok {
		cp.JTI = jti
	}
	if sid, ok := claims["session_id"].(string); ok {
		cp.SID = sid
	}
	if roles, ok := claims["roles"]; ok {
		switch r := roles.(type) {
		case []string:
			cp.Roles = r
		case []interface{}:
			cp.Roles = make([]string, 0, len(r))
			for _, ri := range r {
				if s, ok := ri.(string); ok {
					cp.Roles = append(cp.Roles, s)
				}
			}
		case string:
			// single role maybe
			cp.Roles = []string{r}
		}
	}

	// expiry
	if expv, ok := claims["exp"]; ok {
		switch v := expv.(type) {
		case float64:
			cp.Expiry = time.Unix(int64(v), 0)
		case int64:
			cp.Expiry = time.Unix(v, 0)
		case jsonNumber:
			// fallback (rare)
			if i, err := v.Int64(); err == nil {
				cp.Expiry = time.Unix(i, 0)
			}
		}
	}

	return &cp, nil
}

// helper to satisfy some type assertions (avoid importing encoding/json here).
type jsonNumber interface {
	Int64() (int64, error)
}
