package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token is expired")
)

type PayloadJWT struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type PayloadPaseto struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type VerifiedPayload struct {
	ID        uuid.UUID
	Username  string
	IssuedAt  time.Time
	ExpiredAt time.Time
}

func NewJWTPayload(username string, duration time.Duration) (*PayloadJWT, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	payload := &PayloadJWT{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	return payload, nil
}

func NewPasetoPayload(username string, duration time.Duration) (*PayloadPaseto, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	payload := &PayloadPaseto{
		ID:        tokenID,
		Username:  username,
		IssuedAt:  now,
		ExpiresAt: now.Add(duration),
	}

	return payload, nil
}

func (p *PayloadPaseto) Valid() error {
	if time.Now().After(p.ExpiresAt) {
		return ErrTokenExpired
	}
	return nil
}
