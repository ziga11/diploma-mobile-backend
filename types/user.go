package types

import "time"

type User struct {
	ID          int       `json:"user_id"`
	Group       string    `json:"group"`
	FirstName   string    `json:"firstname"`
	LastName    string    `json:"lastname"`
	Address     string    `json:"address"`
	PhoneNumber string    `json:"mobile"`
	WorkPermit  string    `json:"work_permit"`
	Country     string    `json:"country_name"`
	CountryID   string    `json:"country_id"`
	BirthDate   time.Time `json:"birthdate"`
}

type UserQuery struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
}

type UserLang struct {
	UserID   int    `json:"user_id"`
	LangCode string `json:"lang_code"`
}
