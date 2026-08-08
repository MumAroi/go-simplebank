package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid secret key size: must be at least %d characters", minSecretKeySize)
	}

	return &JWTMaker{secretKey: secretKey}, nil
}

func (m *JWTMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload, err := NewJWTPayload(username, duration)
	if err != nil {
		return "", err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return jwtToken.SignedString([]byte(m.secretKey))
}

func (m *JWTMaker) VerifyToken(tokenString string) (*VerifiedPayload, error) {
	keyFunc := func(jwtToken *jwt.Token) (any, error) {
		return []byte(m.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(tokenString, &PayloadJWT{}, keyFunc, jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Alg(),
	}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*PayloadJWT)
	if !ok || !jwtToken.Valid {
		return nil, ErrInvalidToken
	}

	tokenID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &VerifiedPayload{
		ID:        tokenID,
		Username:  payload.Username,
		IssuedAt:  payload.IssuedAt.Time,
		ExpiresAt: payload.ExpiresAt.Time,
	}, nil
}
