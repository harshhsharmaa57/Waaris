package application

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type Principal struct {
	UserID uuid.UUID
	Email  string
}

type TokenVerifier struct {
	secret []byte
	issuer string
}

func NewTokenVerifier(secret, issuer string) (*TokenVerifier, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if issuer == "" {
		return nil, errors.New("JWT issuer is required")
	}
	return &TokenVerifier{secret: []byte(secret), issuer: issuer}, nil
}

func (v *TokenVerifier) ParseAccessToken(raw string) (Principal, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidAccessToken
		}
		return v.secret, nil
	}, jwt.WithIssuer(v.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Principal{}, ErrInvalidAccessToken
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		return Principal{}, ErrInvalidAccessToken
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		return Principal{}, ErrInvalidAccessToken
	}

	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return Principal{}, ErrInvalidAccessToken
	}

	return Principal{UserID: userID, Email: email}, nil
}
