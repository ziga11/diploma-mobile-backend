package handlers

import (
	"backend/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func fileMimeType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("Error reading file for sniffing: %v", err)
	}

	mimetype := http.DetectContentType(buffer)

	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("Error reseting file pointer: %v", err)
	}

	return mimetype, nil
}

func SaveFile(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	userID := r.FormValue("user_id")
	userIdInt, _ := strconv.Atoi(userID)

	rootFolderID := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
	userFolderID, err := GoogleDrive.GetOrCreateDir(userID, rootFolderID)
	if err != nil {
		http.Error(w, "Error managing user folder: "+err.Error(), http.StatusInternalServerError)
		return
	}

	mimetype, err := fileMimeType(file)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	driveFile, err := GoogleDrive.UploadFile(
		file,
		fileHeader.Filename,
		mimetype,
		userFolderID)
	if err != nil {
		http.Error(w, "Upload to Drive failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	doc, err := InsertDocument(types.Document{
		UserID:   userIdInt,
		Title:    r.FormValue("title"),
		Type:     r.FormValue("type"),
		Mimetype: &mimetype,
		DriveID:  driveFile.Id,
	})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(doc)
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var doc types.Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	query := "DELETE FROM mobile.document WHERE id = $1 RETURNING drive_id"

	var driveID string
	err := MobileDB.QueryRow(query, doc.ID).Scan(&driveID)
	if err != nil {
		http.Error(w, "DB Delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if driveID != "" {
		GoogleDrive.TrashDir(driveID)
	}

	w.WriteHeader(http.StatusOK)
}

func FetchFile(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	docId, _ := getQueryParam[int](w, r, "id")

	var driveID string
	err := MobileDB.QueryRow("SELECT drive_id FROM mobile.document WHERE id = $1", docId).Scan(&driveID)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	driveID = strings.TrimSpace(driveID)

	res, err := GoogleDrive.srv.Files.Get(driveID).Download()
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "Failed to download from Drive: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", res.Header.Get("Content-Length"))

	io.Copy(w, res.Body)
}

func InsertDocument(doc types.Document) (types.Document, error) {
	var docID int
	query := `INSERT INTO mobile.document
			(user_id, drive_id, title, type, mimetype) VALUES
			($1, $2, $3, $4, $5)
		 RETURNING id`

	err := MobileDB.QueryRow(query, doc.UserID, doc.DriveID, doc.Title, doc.Type, doc.Mimetype).Scan(&docID)
	if err != nil {
		log.Printf("Error inserting document %v %d\n", err.Error(), http.StatusInternalServerError)
		return types.Document{}, err
	}

	doc.ID = docID

	return doc, err
}

func UserDocs(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	docId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}

	var documents []types.Document

	rows, err := MobileDB.Query(`
			SELECT
				id,
				user_id,
				COALESCE(title, '') as title,
				COALESCE(type, '') as type,
				drive_id,
				mimetype,
				date
			FROM mobile.document
			WHERE user_id = $1
			ORDER BY date DESC`, docId)
	if err != nil {
		http.Error(w, "Failed to query database: "+err.Error(), http.StatusNotFound)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var doc types.Document
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.Title, &doc.Type, &doc.DriveID, &doc.Mimetype, &doc.Date); err != nil {
			http.Error(w, "Failed to read row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		documents = append(documents, doc)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, "Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(documents); err != nil {
		http.Error(w, "Failed to encode documents: "+err.Error(), http.StatusInternalServerError)
	}
}

func FetchDocById(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	docId, ok := getQueryParam[int](w, r, "id")
	if !ok {
		return
	}

	query := `SELECT
			user_id,
			COALESCE(title, '') as title,
			COALESCE(type, '') as type,
			drive_id,
			mimetype
		FROM mobile.document
		WHERE id = $1`

	var doc types.Document
	doc.ID = docId

	err := MobileDB.QueryRow(query, docId).Scan(&doc.UserID, &doc.Title, &doc.Type, &doc.DriveID, &doc.Mimetype)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		http.Error(w, "Failed to encode document: "+err.Error(), http.StatusInternalServerError)
	}
}
