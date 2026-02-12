package types

import "time"

type HashedToken struct {
	AccountID int       `json:"account_id"`
	Token     *string   `json:"token"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	Email     *string   `json:"email"`
}

type EncryptedToken struct {
	AccountID int       `json:"account_id"`
	Token     string    `json:"token"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	Nonce     *string
}

type HashedTokenLookup struct {
	AccountID *int
	Type      *string
	Token     *string
}

type EncryptedTokenLookup struct {
	AccountID   *int
	Type        *string
	Token       *[]byte
	ExpiredOnly bool
}
