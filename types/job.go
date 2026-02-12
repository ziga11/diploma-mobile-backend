package types

import (
	"time"
)

type UserJob struct {
	UserID    int        `json:"user_id"`
	JobID     int        `json:"job_id"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type UserJobCompany struct {
	CompanyID   int        `json:"company_id"`
	CompanyName string     `json:"company_name"`
	UserID      int        `json:"user_id"`
	Workpermit  string     `json:"workpermit"`
	JobTitle    string     `json:"job_title"`
	JobID       int        `json:"job_id"`
	Status      string     `json:"status"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

type JobExternalDocuments struct {
	ID           int     `json:"id"`
	ObligationID int     `json:"obligation_id"`
	JobID        int     `json:"job_id"`
	DownloadUrl  *string `json:"download_url"`
}

type Job struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	CompanyID int    `json:"company_id"`
}
