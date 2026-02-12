package types

import "time"

type Account struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Email     string    `json:"email"`
	LangCode  string    `json:"lang_code"`
	CreatedAt time.Time `json:"created_at"`
}

type NewAccount struct {
	UserID   int
	Email    string
	LangCode string
}

type ClientLogin struct {
	Email string `json:"email"`
	PW    string `json:"pw"`
}

type SetPW struct {
	AccountID *int    `json:"account_id"`
	Token     *string `json:"raw_token"`
	RawPW     string  `json:"pw"`
}

type ResetPW struct {
	Email string `json:"email"`
	Token string `json:"token"`
	PW    string `json:"pw"`
}

type AccountQuery struct {
	AccountID int    `json:"account_id"`
	Email     string `json:"email"`
}
