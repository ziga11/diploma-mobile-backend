package handlers

import (
	"backend/types"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const InfoUserID = 1

func GetUser(userId int) (types.User, error) {
	var user types.User

	var query = `
		SELECT 
			u.id,
			firstName,
			lastName,
			address,
			c.name,
			birthdate,
			workpermit,
			COALESCE(Mobile, '') AS mobile,
			COALESCE(g.name, '') AS group
		FROM mobile.users u
		LEFT JOIN mobile.user_group_link ugl
			ON u.id = ugl.user_id
		LEFT JOIN mobile.groups g
			ON ugl.group_id = g.id
		LEFT JOIN mobile.country c
			ON c.id = u.country_id
		WHERE u.id = $1`

	err := MobileDB.QueryRow(query, userId).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Country,
		&user.BirthDate,
		&user.WorkPermit,
		&user.PhoneNumber,
		&user.Group)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found %d", http.StatusNotFound)
		} else {
			log.Printf("%v %d\n", err.Error(), http.StatusInternalServerError)
		}
		return types.User{}, fmt.Errorf("Failed to fetch user (%v): %v", userId, err)
	}

	return user, nil
}

func GetUserGroup(userId int) (types.Group, error) {
	var group types.Group

	query := `SELECT g.id, g.name
			FROM mobile.groups g
			JOIN mobile.user_group_link ugl ON
			    g.id = ugl.group_id
			WHERE
			    ugl.user_id = $1`

	err := MobileDB.QueryRow(query, userId).Scan(&group.ID, &group.Name)
	if err != nil {
		return types.Group{}, err
	}

	return group, err
}

func UpdateUserData(userId int, dataType, data string) {
	query := fmt.Sprintf("UPDATE mobile.users SET %s = $1 WHERE id = $2", dataType)
	_, err := MobileDB.Exec(query, data, userId)
	if err != nil {
		log.Printf("Error changing user's data --> %v", err)
		return
	}
}

func UpdateUserReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err := UpdateUser(req)
	RespondIfErr(w, err, http.StatusBadRequest)
}

func InsertUser(tx *sql.Tx, u types.User) (int, error) {
	var userId int

	userQuery := `
		INSERT INTO mobile.users
		(firstname, lastname, birthdate, country_id, address)
		    SELECT $1, $2, $3, c.id, $5
			FROM mobile.country c WHERE name ILIKE $4
		RETURNING id`

	err := tx.QueryRow(
		userQuery,
		u.FirstName,
		u.LastName,
		u.BirthDate,
		u.Country,
		u.Address,
	).Scan(&userId)
	if err != nil {
		log.Printf("Error inserting user: %v\n", err)
		return -1, fmt.Errorf("Napak pri vnosu uporabnika v podatkovno bazo")
	}

	return userId, nil
}

func UpdateUser(u types.User) (int, error) {
	if u.ID <= 1 {
		return -1, fmt.Errorf("userID is invalid... not updating user")
	}
	var userId int

	userQuery := `
		UPDATE mobile.users SET
			firstname = $2,
			lastname = $3,
			birthdate = $4,
			country_id = (SELECT id FROM mobile.country WHERE name = $5),
			address = $6
		WHERE id = $1
		RETURNING id`

	err := MobileDB.QueryRow(
		userQuery,
		u.ID,
		u.FirstName,
		u.LastName,
		u.BirthDate,
		u.Country,
		u.Address,
	).Scan(&userId)
	if err != nil {
		log.Printf("Error updating user: %v\n", err)
		return -1, fmt.Errorf("Failed to update user: %v", err)
	}

	return userId, nil
}

func FetchUsers(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	searchQuery, err := QueryGetType[string](r, "search")
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	users, err := SearchUsersInDB(searchQuery)
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(users)
}

func SearchUsersInDB(search string) ([]types.User, error) {
	query := `SELECT 
			u.id,
			firstName,
			lastName,
			address,
			c.id,
			c.name,
			birthDate,
			workPermit,
			COALESCE(mobile, '')
		FROM mobile.users u
		LEFT JOIN mobile.country c ON
			c.id = u.country_id
		WHERE
		    CONCAT(firstName, ' ', lastName) ILIKE $1`
	rows, err := MobileDB.Query(query, "%"+search+"%")
	if err != nil {
		fmt.Printf("Query execution error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var u types.User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Address, &u.CountryID, &u.Country, &u.BirthDate, &u.WorkPermit, &u.PhoneNumber); err != nil {
			fmt.Printf("Failed to scan row: %v", err)
			return nil, err
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration: %v", err)
		return nil, err
	}

	return users, nil
}

func UserById(w http.ResponseWriter, r *http.Request) {
	aId, err := headerAccId(r)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	uId, err := userIDByAccID(aId)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	userId, err := QueryGetType[int](r, "user_id")
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	if !AuthToken(r) || (userId != InfoUserID && uId != userId) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := GetUser(userId)
	if RespondIfErr(w, err, http.StatusNotFound) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode user: "+err.Error(), http.StatusInternalServerError)
	}
}

func userIDByAccID(accId int) (int, error) {
	query := `SELECT
			u.id
		FROM mobile.users u
		LEFT JOIN mobile.accounts a ON
			u.id = a.user_id
		WHERE a.id = $1
		`

	var userId int

	err := MobileDB.QueryRow(query, accId).Scan(&userId)
	if err != nil {
		log.Printf("Failed to get UserId by AccId %v", err)
		return -1, fmt.Errorf("Failed to get UserId by AccId: %v", err)
	}

	return userId, nil
}

func FetchAllUsers() []types.User {
	query := `SELECT
			u.id,
			firstName,
			lastName,
			address,
			c.id,
			c.name,
			birthDate,
			workPermit,
			COALESCE(mobile, '')
		FROM mobile.users u
		LEFT JOIN mobile.country c ON
			c.id = u.country_id
		`

	rows, err := MobileDB.Query(query)
	if err != nil {
		fmt.Printf("Query execution error: %v", err)
		return nil
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var u types.User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Address, &u.CountryID, &u.Country, &u.BirthDate, &u.WorkPermit, &u.PhoneNumber); err != nil {
			fmt.Printf("Failed to scan row: %v", err)
			return nil
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration: %v", err)
		return nil
	}

	return users
}
