package types

import "time"

type Document struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	DriveID  string    `json:"drive_id"`
	Title    string    `json:"title"`
	Type     string    `json:"type"`
	Mimetype *string   `json:"mimetype"`
	Date     time.Time `json:"date"`
}
