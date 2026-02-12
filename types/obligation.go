package types

import (
	"time"
)

type Obligation struct {
	Id           int  `json:"id"`
	Uploadable   bool `json:"is_uploadable"`
	HasTextField bool `json:"has_text_field"`
	ExampleDocID *int `json:"example_doc_id"`

	Translations Translations `json:"translations"`

	UserID    int       `json:"user_id"`
	Status    string    `json:"status"`
	Reasoning string    `json:"reasoning"`
	Date      time.Time `json:"date"`

	UploadedDocIDs []int  `json:"uploaded_doc_ids"`
	TextValue      string `json:"text_value"`

	GoogleFileID *string `json:"google_file_id"`
}

type Oblg struct {
	ID           int  `json:"id"`
	ExampleDocID *int `json:"example_doc_id"`
	Uploadable   bool `json:"uplodable"`
	TextField    bool `json:"text_field"`
}

type ObligationApplicability struct {
	ObligationID     int
	CountryIDs       []int    `json:"country_id"`
	JobIDs           []int    `json:"job_ids"`
	WorkpermitStatus []string `json:"workpermit_status"`
}

type ObligationTranslation struct {
	ObligationID int
	Translations Translations `json:"translations"`
}

type UserObligation struct {
	ObligationID int       `json:"obligation_id"`
	UserId       int       `json:"user_id"`
	Date         time.Time `json:"date"`
	Status       string    `json:"status"`
	Reasoning    string    `json:"reasoning"`
	TextField    *string   `json:"text_field"`
}

type UserObligationDoc struct {
	UODocID        int
	LinkId         int       `json:"uol_id"`
	UploadedDocIDs []int     `json:"uploaded_doc_ids"`
	UODDate        time.Time `json:"doc_date"`
}

type CreateObligation struct {
	Id               *int
	Uploadable       bool
	CountryIDs       *[]int
	CompanyIDs       *[]int
	WorkpermitStatus *[]string
	Translations     Translations
}

type AssignObligationRequest struct {
	UserId           int    `json:"user_id"`
	Country          string `json:"country"`
	WorkpermitStatus string `json:"workpermit_status"`
}

type AssignObligation struct {
	UserJobCompany   *UserJobCompany `json:"user_job_company"`
	Country          *string         `json:"country"`
	WorkpermitStatus string          `json:"workpermit_status"`
}

type ObligationResponse struct {
	ObligationID int     `json:"obligation_id"`
	Status       *string `json:"status"`
	UserID       int     `json:"user_id"`
	DocIDs       []int   `json:"doc_ids"`
	TextField    *string `json:"text_value"`
}
