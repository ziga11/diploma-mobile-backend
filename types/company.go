package types

import (
	"time"
)

type UserCompany struct {
	CompanyID int        `json:"id"`
	UserID    int        `json:"user_id"`
	Name      string     `json:"name"`
	Position  string     `json:"position"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type Company struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
