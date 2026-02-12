package handlers

import (
	"backend/types"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func JobFromPosAndCompany(job *types.Job) (types.Job, error) {
	query := `SELECT
			id,
			title,
			company_id
		  FROM mobile.job
		  WHERE
			title ILIKE $1 || '%' AND
			company_id = $2`

	err := MobileDB.QueryRow(query, job.Title, job.CompanyID).Scan(&job.ID, &job.Title, &job.CompanyID)
	if err != nil {
		log.Printf("Failed to select job by position and company: %v", err)
		return types.Job{}, fmt.Errorf("Delovno mesto pri tem podjetju ne obstaja")
	}

	return *job, nil
}

func UpdateUserJobStatus(userId int, status string) error {
	query := `UPDATE mobile.user_job
			SET status = $2
		  WHERE
			user_id = $1
		`

	_, err := MobileDB.Exec(query, userId, status)
	if err != nil {
		return fmt.Errorf("Failed to update user job status: %v", err)
	}

	return nil
}

func UserJobDetailsReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}

	userJobCompany, err := UserJobDetails(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "No assigned job", http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(userJobCompany); err != nil {
		http.Error(w, "Failed to encode company data: "+err.Error(), http.StatusInternalServerError)
	}
}

func UserJobDetails(userId int) (types.UserJobCompany, error) {
	query := `SELECT
			c.id,
			c.name,
			u.id,
			u.workpermit,
			j.id,
			j.title,
			uj.status,
			uj.start_date,
			uj.end_date
		FROM mobile.user_job uj
		JOIN mobile.job j 
			ON j.id = uj.job_id
		JOIN mobile.users u
			ON u.id = uj.user_id
		JOIN mobile.company c
			ON c.id = j.company_id
		WHERE user_id = $1`

	var ujc types.UserJobCompany
	err := MobileDB.QueryRow(query, userId).Scan(
		&ujc.CompanyID,
		&ujc.CompanyName,
		&ujc.UserID,
		&ujc.Workpermit,
		&ujc.JobID,
		&ujc.JobTitle,
		&ujc.Status,
		&ujc.StartDate,
		&ujc.EndDate)
	if err != nil {
		log.Printf("failed to fetch job details of %v --> %v", userId, err)
		return types.UserJobCompany{}, err
	}

	return ujc, nil
}

func insertUserJob(tx *sql.Tx, job types.UserJob) error {
	query := `INSERT INTO mobile.user_job(
			job_id,
			user_id,
			start_date,
			end_date
		) VALUES ($1, $2, $3, $4)`

	_, err := tx.Exec(query, job.JobID, job.UserID, job.StartDate, job.EndDate)
	if err != nil {
		log.Printf("Failed to insert user_job: %v", err)
		return fmt.Errorf("Napaka pri vnosu povezave med delovnim mestom in uporabnikom")
	}

	return nil
}

func JobList(ctx context.Context) []types.UserJobCompany {
	query := `SELECT
			j.id,
			j.title,
			c.name
		 FROM mobile.job j
		 LEFT JOIN mobile.company c ON
			j.company_id = c.id`

	rows, err := MobileDB.QueryContext(ctx, query)
	if err != nil {
		log.Println("Failed to list jobs: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	var ujcs []types.UserJobCompany

	for rows.Next() {
		var ujc types.UserJobCompany
		if err := rows.Scan(
			&ujc.JobID,
			&ujc.JobTitle,
			&ujc.CompanyName); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil
		}

		ujcs = append(ujcs, ujc)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	return ujcs
}
