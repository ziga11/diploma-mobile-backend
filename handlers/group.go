package handlers

import (
	"backend/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func UpdateUserGroup(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.UserGroup
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request --> sent Register: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err := SetUserGroup(req)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func SetUserGroup(ug types.UserGroup) error {
	if ug.GroupID == nil {
		groupId, err := GetGroupIdByName(ug.GroupName)
		if err != nil {
			return err
		}

		ug.GroupID = &groupId
	}

	log.Printf("Updating user group: %v --> %v for user_id: %v", ug.GroupName, *ug.GroupID, *ug.UserID)

	_, err := MobileDB.Exec(`
		UPDATE mobile.user_group_link
			SET group_id = $2
		WHERE
			user_id = $1`,
		ug.UserID, ug.GroupID)
	if err != nil {
		log.Printf("Failed to insert or update the user group: %v", err)
		return fmt.Errorf("Failed to insert or update the user group: %v", err)
	}

	return err
}

func GetGroupIdByName(gName string) (int, error) {
	query := `
		SELECT id 
		FROM mobile.groups 
		WHERE
			name ILIKE '%' || $1 || '%' 
		    LIMIT 1
        `
	var groupId int

	err := MobileDB.QueryRow(query, gName).Scan(&groupId)
	if err != nil {
		return -1, fmt.Errorf("Failed to scan groupId by name: %v", err)
	}

	return groupId, nil
}

func ListOfGroups(ctx context.Context) []types.Group {
	tx, err := MobileDB.Begin()
	if err != nil {
		log.Println("Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	query := "SELECT id, name FROM mobile.groups"

	rows, err := tx.Query(query)
	if err != nil {
		log.Println("Failed to list groups: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	var groups []types.Group

	for rows.Next() {
		var group types.Group
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil
		}

		groups = append(groups, group)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed to commit the transaction %v --> ListOfGroups", err)
		return nil
	}

	return groups
}
