package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type Principal struct {
	UserID uuid.UUID
	Email  string
}

type TokenManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	now       func() time.Time
}

func NewTokenManager(secret, issuer string, accessTTL time.Duration) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if accessTTL <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &TokenManager{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL, now: time.Now}, nil
}

func (m *TokenManager) IssueAccessToken(user domain.User) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.accessTTL)
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"iss":   m.issuer,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, expiresAt, err
}

func (m *TokenManager) ParseAccessToken(raw string) (Principal, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidAccessToken
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Principal{}, ErrInvalidAccessToken
	}
	subject, ok := claims["sub"].(string)
	if !ok {
		return Principal{}, ErrInvalidAccessToken
	}
	id, err := uuid.Parse(subject)
	if err != nil {
		return Principal{}, ErrInvalidAccessToken
	}
	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return Principal{}, ErrInvalidAccessToken
	}
	return Principal{UserID: id, Email: email}, nil
}

func NewRefreshToken(userID uuid.UUID, ttl time.Duration, now time.Time) (string, domain.RefreshToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", domain.RefreshToken{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	return value, domain.RefreshToken{ID: uuid.New(), UserID: userID, Hash: HashRefreshToken(value), CreatedAt: now.UTC(), ExpiresAt: now.UTC().Add(ttl)}, nil
}

func HashRefreshToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
