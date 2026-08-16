package token

import "time"

type Maker interface {
	CreateToken(username string, duration time.Duration) (string, *VerifiedPayload, error)
	VerifyToken(token string) (*VerifiedPayload, error)
}
