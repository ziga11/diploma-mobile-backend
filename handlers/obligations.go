package handlers

import (
	"backend/types"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
)

func CheckCompletionStatus(userID int) bool {
	query := `SELECT 
		    COUNT(*) > 0
		    AND COUNT(*) FILTER (WHERE status = 'Completed') = COUNT(*)
		FROM mobile.user_obligations
		WHERE user_id = $1;`

	var allCompleted bool
	err := MobileDB.QueryRow(query, userID).Scan(&allCompleted)
	if err != nil {
		log.Printf("AllObligationsCompleted err: %v", err)
		return false
	}

	return allCompleted
}

func UserObligationsReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}

	obligations, err := UserObligations(types.UserLang{UserID: userId})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(obligations); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		fmt.Printf("Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func UserObligations(ul types.UserLang) ([]types.Obligation, error) {
	query := `SELECT * FROM mobile.fetch_user_obligations($1)`

	rows, err := MobileDB.Query(query, ul.UserID)
	if err != nil {
		log.Printf("Query execution error: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Failed to query for assgined obligations: %v", err)
	}
	defer rows.Close()

	obligations := []types.Obligation{}
	for rows.Next() {
		var obls types.Obligation
		var uploadedDocIDs []int64

		if err := rows.Scan(
			&obls.Id,
			&obls.UserID,
			&obls.Translations,
			&obls.ExampleDocID,
			&obls.Uploadable,
			&obls.HasTextField,
			&obls.GoogleFileID,
			pq.Array(&uploadedDocIDs),
			&obls.Status,
			&obls.Reasoning,
			&obls.Date,
			&obls.TextValue,
		); err != nil {
			log.Printf("Failed to scan row: "+err.Error(), http.StatusInternalServerError)
			return nil, fmt.Errorf("Failed to scan obligation row: %v", err)
		}

		obls.UploadedDocIDs = make([]int, len(uploadedDocIDs))
		for i, v := range uploadedDocIDs {
			obls.UploadedDocIDs[i] = int(v)
		}

		obligations = append(obligations, obls)
	}
	if err = rows.Err(); err != nil {
		fmt.Printf("Error during row iterations: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Failed during row interatiors: %v", err)
	}

	return obligations, nil
}

func CreateEntireObligation(o types.Oblg, ot types.ObligationTranslation, oa types.ObligationApplicability) error {
	tx, err := MobileDB.Begin()
	if err != nil {
		log.Printf("Failed to start transaction")
		return err
	}
	defer tx.Rollback()

	oId, err := CreateObligation(tx, o)
	if err != nil {
		return err
	}

	ot.ObligationID = oId
	oa.ObligationID = oId

	err = CreateObligationTranslations(tx, ot)
	if err != nil {
		return err
	}

	err = CreateObligationApplicability(tx, oa)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Failed to commit transaction:", err)
		return err
	}

	return nil
}

func CreateObligation(tx *sql.Tx, o types.Oblg) (int, error) {
	var id int
	err := tx.QueryRow(`
		INSERT INTO
		mobile.obligation(
			uploadable,
			text_field
		) VALUES ($1, $2)
		RETURNING id`,
		o.Uploadable, o.TextField).Scan(&id)

	if err != nil {
		log.Printf("Failed to insert obligation: %v", err)
		return -1, fmt.Errorf("Failed to insert obligation: %v", err)
	}

	return id, nil
}

func CreateObligationTranslations(tx *sql.Tx, o types.ObligationTranslation) error {
	query := `INSERT INTO
			mobile.obligation_translations(obligation_id, language_code, title, body)
			    VALUES ($1, $2, $3, $4)`

	entries := []struct {
		Code string
		Data types.LangContent
	}{
		{"si", o.Translations.Si},
		{"en", o.Translations.En},
		{"bs", o.Translations.Bs},
	}

	for _, entry := range entries {
		_, err := tx.Exec(query, o.ObligationID, entry.Code, entry.Data.Title, entry.Data.Body)
		if err != nil {
			log.Printf("Failed to insert obligation translations: %v\n", err)
			return fmt.Errorf("Failed to insert obligation translations: %v\n", err)
		}
	}

	return nil
}

func CreateObligationApplicability(tx *sql.Tx, o types.ObligationApplicability) error {
	countryIds := []int{0}
	if len(o.CountryIDs) > 0 {
		countryIds = o.CountryIDs
	}

	jobIds := []int{0}
	if len(o.JobIDs) > 0 {
		jobIds = o.JobIDs
	}

	workpermitStatuses := []string{""}
	if len(o.WorkpermitStatus) > 0 {
		workpermitStatuses = o.WorkpermitStatus
	}

	stmt, err := tx.Prepare(
		`INSERT INTO mobile.obligation_applicability(
			obligation_id,
			country_id,
			job_id,
			workpermit_status
		) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		log.Println("Failed to prepare statement: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	defer stmt.Close()

	for _, countryID := range countryIds {
		for _, jobID := range jobIds {
			for _, status := range workpermitStatuses {
				var countryIDPtr, jobIdPtr *int
				var statusPtr *string

				if countryID != 0 {
					countryIDPtr = &countryID
				}
				if jobID != 0 {
					jobIdPtr = &jobID
				}

				if status != "" {
					statusPtr = &status
				}

				if _, err := stmt.Exec(o.ObligationID, countryIDPtr, jobIdPtr, statusPtr); err != nil {
					log.Println("Failed to insert obligation_translation: "+err.Error(), http.StatusInternalServerError)
					return err
				}
			}
		}
	}

	return nil
}

func AddExampleDocToObligation(o types.Oblg) error {
	_, err := MobileDB.Exec("UPDATE mobile.Obligation SET document_id = $1 WHERE id = $2", o.ExampleDocID, o.ID)
	if err != nil {
		log.Println("Failed to add docID to obligation: "+err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func SetUserObligationStatus(uo *types.UserObligation) (int, error) {
	var userObligationId int

	err := MobileDB.QueryRow(`
		UPDATE mobile.user_obligations
		    SET
			status = $1,
			text_value = $2
		    WHERE
			user_id = $3 AND
			obligation_id = $4
		    RETURNING id`,
		uo.Status, uo.TextField, uo.UserId, uo.ObligationID).Scan(&userObligationId)

	if err != nil {
		log.Println("Failed to set status to user obligations: "+err.Error(), http.StatusInternalServerError)
		return -1, fmt.Errorf("Failed to set status to user obligations: %v", err.Error())
	}

	return userObligationId, nil
}

func AllObligationCompleted(userId int) error {
	group, err := GetUserGroup(userId)
	if err != nil {
		return err
	}

	groupName := strings.ToLower(group.Name)

	if !slices.Contains([]string{"tujina", "napoteni delavec"}, groupName) {
		return err
	}

	userJob, err := UserJobDetails(userId)
	if err != nil {
		return err
	}

	switch groupName {
	case "tujina":
		RemoveAssignedObligations(userId)
		err := AssignObligations(types.AssignObligation{
			UserJobCompany:   &userJob,
			WorkpermitStatus: TernaryOperator(userJob.Workpermit == "Permanent", "HAS", "TEMPORARY"),
		})
		if err != nil {
			return err
		}

		err = SetUserGroup(types.UserGroup{UserID: &userId, GroupName: "Napoteni Delavec"})
		if err != nil {
			return err
		}
	case "napoteni delavec":
		RemoveAssignedObligations(userId)
		err := SetUserGroup(types.UserGroup{UserID: &userId, GroupName: "Zaposlen"})
		if err != nil {
			return err
		}
	}

	return nil
}

func SetObligationStatusReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.ObligationResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	uolId, err := SetUserObligationStatus(&types.UserObligation{
		UserId:       req.UserID,
		ObligationID: req.ObligationID,
		Status:       *req.Status,
		TextField:    req.TextField,
	})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	if req.Status != nil && *req.Status == "Incomplete" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if CheckCompletionStatus(req.UserID) {
		AllObligationCompleted(req.UserID)
	}

	if req.TextField != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = AddDocsToObligation(uolId, req.DocIDs)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode("Added doc to userobligation"); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func AddDocsToObligation(uolId int, docIds []int) error {
	if len(docIds) == 0 {
		return nil
	}
	query := "INSERT INTO mobile.user_obligation_documents(link_id, document_id) VALUES ($1, $2)"

	for _, docId := range docIds {
		_, err := MobileDB.Exec(query, uolId, docId)
		if err != nil {
			return fmt.Errorf("Failed to update userobligationlink :%v", err)
		}
	}

	return nil
}

func RemoveUserObligationDocs(userId, obligationID int) error {
	query := `DELETE FROM user_obligation_document
		  WHERE
			user_id = $1 AND
			obligation_id = $2`
	_, err := MobileDB.Exec(query, userId, obligationID)
	if err != nil {
		return fmt.Errorf("Failed to delete user obligation documents: %v", err)
	}

	return nil
}

func RemoveUserObligationDocsReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.UserObligation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	err := RemoveUserObligationDocs(req.UserId, req.ObligationID)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode("Inserted the obligation link to the user"); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func AssignObligations(req types.AssignObligation) error {
	query := `INSERT INTO mobile.user_obligations(user_id, obligation_id)
			SELECT $1, oa.obligation_id
			FROM mobile.obligation_applicability oa
			WHERE
			    oa.workpermit_status = $4 AND
			    (
				oa.country_id IS NULL OR
			    	oa.country_id = (SELECT id FROM mobile.country where LOWER(name) = LOWER($2))
			    ) AND
			    (oa.job_id IS NULL OR oa.job_id = $3)
			ON CONFLICT(obligation_id, user_id)
			    DO NOTHING;`

	_, err := MobileDB.Exec(query, req.UserJobCompany.UserID, req.Country, req.UserJobCompany.JobID, req.WorkpermitStatus)
	if err != nil {
		log.Printf("Failed to assign obligations: "+err.Error(), http.StatusInternalServerError)
		return fmt.Errorf("Failed to assign obligations: %v", err)
	}

	return nil
}

func AssignObligationsReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	var req types.AssignObligationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userJob, err := UserJobDetails(req.UserId)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	if err := AssignObligations(types.AssignObligation{
		UserJobCompany:   &userJob,
		Country:          &req.Country,
		WorkpermitStatus: req.WorkpermitStatus,
	}); err != nil {
		http.Error(w, "Failed to assign obligations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode("Inserted the obligation link to the user"); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func AssignSingleObligation(obligation types.UserObligation) {
	query := "INSERT INTO mobile.user_obligations(user_id, obligation_id) VALUES($1, $2)"

	_, err := MobileDB.Exec(query,
		fmt.Sprintf("%d", obligation.UserId), fmt.Sprintf("%d", obligation.ObligationID))
	if err != nil {
		log.Println("Failed to assign obligation: " + err.Error())
	}
}

func FetchObligationByID(w http.ResponseWriter, r *http.Request) {
	oId, ok := getQueryParam[int](w, r, "id")
	if !ok {
		return
	}

	query := `
		SELECT
			o.id,
			jsonb_object_agg(
			    ot.language_code,
			    jsonb_build_object(
				'title', ot.title,
				'body', ot.body
			    )
			) AS translations,
			o.document_id
		FROM mobile.obligation o
		LEFT JOIN mobile.obligation_translations ot
			ON o.id = ot.obligation_id
		WHERE
			o.id = $1
		GROUP BY
			o.id, o.document_id
		`

	obligation := types.Obligation{UserID: 0, Status: "", Reasoning: "", Date: time.Now()}

	err := MobileDB.QueryRow(query, oId).Scan(&obligation.Id, &obligation.Translations, &obligation.ExampleDocID)
	if err != nil {
		http.Error(w, "Failed to fetch obligation by ID "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(obligation); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func RemoveAssignedObligations(userId int) {
	tx, err := MobileDB.Begin()
	if err != nil {
		log.Println("Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM mobile.user_obligations WHERE user_id = $1", userId)
	if err != nil {
		log.Printf("Error deleting user_obligations: %v", err)
	}
}

func RemoveSingleAssignedObligation(userId, obligationId int) {
	_, err := MobileDB.Exec("DELETE FROM user_obligations WHERE user_id = $1 AND obligation_id = $2", userId, obligationId)
	if err != nil {
		log.Printf("Error deleting userobligationlink: %v", err)
	}
}

func DistinctObligations(ctx context.Context) []types.ObligationTranslation {
	var obligations []types.ObligationTranslation
	query := `SELECT DISTINCT
			obligation_id, 
			jsonb_object_agg(
			    language_code,
			    jsonb_build_object(
				'title', title,
				'body', body
			    )
			) AS translations
			FROM mobile.obligation_translations
			GROUP BY
			    obligation_id
		`

	rows, err := MobileDB.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Query execution error: "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var obligation types.ObligationTranslation
		if err := rows.Scan(&obligation.ObligationID, &obligation.Translations); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil
		}
		obligations = append(obligations, obligation)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error during row iterations: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	return obligations
}

func DeleteObligation(id int) {
	query := `
        DELETE FROM mobile.obligation
	   WHERE id = $1`

	_, err := MobileDB.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete obligation: %v\n", err)
	}
}

func ObligationReminders() {
	var obligations []types.Obligation

	query := `
		SELECT
		    o.id,
		    jsonb_object_agg(
			ot.language_code,
			jsonb_build_object(
			    'title', ot.title,
			    'body', ot.body
			)
		    ) AS translations,
		    ub.user_id,
		    ub.date
		FROM mobile.obligation o
		LEFT JOIN mobile.user_obligations ub
		    ON ub.obligation_id = o.id
		LEFT JOIN mobile.users u
		    ON u.id = ub.user_id
		LEFT JOIN mobile.obligation_translations ot
		    ON o.id = ot.obligation_id
		WHERE
		    ub.status = 'Incomplete'
		GROUP BY
		    o.id,
		    ub.user_id,
		    ub.date`

	rows, err := MobileDB.Query(query)
	if err != nil {
		log.Println("Failed to fetch obligations:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var o types.Obligation
		if err := rows.Scan(&o.Id, &o.Translations, &o.UserID, &o.Date); err != nil {
			log.Println("scan error:", err)
			return
		}
		obligations = append(obligations, o)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, o := range obligations {
		sendObligationReminder(o)
	}
}
