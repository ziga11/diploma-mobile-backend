package handlers

import (
	"backend/types"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var boardRowMap = map[string]string{
	"Firstname":     "120",
	"Lastname":      "121",
	"Phone":         "122",
	"Email":         "123",
	"DateOfBirth":   "124",
	"Citizenship":   "125",
	"Address":       "126",
	"Company":       "128",
	"Position":      "129",
	"StartContract": "130",
	"EndContract":   "131",
}

type processedRequest struct {
	Account types.NewAccount
	User    types.User
	Job     types.UserJob
}

func validateRows(rows map[string]types.Row, keys []string) error {
	for _, k := range keys {
		id := boardRowMap[k]
		if rows[id].Entry.Value == nil {
			boardField := rows[id].Field.Name
			fieldName := TernaryOperator(boardField == nil, "", *boardField)

			log.Printf("nil value for %v", k)

			return fmt.Errorf("Ni vnosa za stolpec %s(%s)", fieldName, id)
		}
	}
	return nil
}

func processUser(rows map[string]types.Row) (types.User, error) {
	fields := []string{"Firstname", "Lastname", "Phone", "DateOfBirth", "Citizenship", "Address"}
	err := validateRows(rows, fields)
	if err != nil {
		return types.User{}, err
	}

	user := types.User{
		FirstName:   *rows[boardRowMap["Firstname"]].Entry.Value,
		LastName:    *rows[boardRowMap["Lastname"]].Entry.Value,
		BirthDate:   *ParseTimeDate(*rows[boardRowMap["DateOfBirth"]].Entry.Value),
		PhoneNumber: *rows[boardRowMap["Phone"]].Entry.Value,
		Country:     *rows[boardRowMap["Citizenship"]].Entry.Value,
		Address:     *rows[boardRowMap["Address"]].Entry.Value,
	}

	return user, nil
}

func processAccount(rows map[string]types.Row) (types.NewAccount, error) {
	emailEntry := rows[boardRowMap["Email"]].Entry
	if emailEntry.Value == nil {
		return types.NewAccount{}, fmt.Errorf("Email uporabnika ni vnesen")
	}

	var acc types.NewAccount
	acc = types.NewAccount{
		Email: *emailEntry.Value,
	}

	return acc, nil
}

func processJob(rows map[string]types.Row) (types.UserJob, error) {
	fields := []string{"Company", "Position", "StartContract", "EndContract"}
	err := validateRows(rows, fields)
	if err != nil {
		return types.UserJob{}, err
	}

	company, err := CompanyByName(*rows[boardRowMap["Company"]].Entry.Value)
	if err != nil {
		log.Printf("Failed to process job, non existing company: %v", err)
		return types.UserJob{}, fmt.Errorf("Napaka pri procesiranju delovnega mesta, podjetje ne obstaja")
	}

	job, err := JobFromPosAndCompany(
		&types.Job{
			CompanyID: company.ID,
			Title:     *rows[boardRowMap["Position"]].Entry.Value,
		})
	if err != nil {
		return types.UserJob{}, err
	}

	return types.UserJob{
		JobID:     job.ID,
		StartDate: ParseTimeDate(*rows[boardRowMap["StartContract"]].Entry.Value),
		EndDate:   ParseTimeDate(*rows[boardRowMap["EndContract"]].Entry.Value),
	}, nil
}

func processRequestRows(rows map[string]types.Row) (processedRequest, error) {
	user, err := processUser(rows)
	if err != nil {
		return processedRequest{}, err
	}

	acc, err := processAccount(rows)
	if err != nil {
		return processedRequest{}, err
	}

	job, err := processJob(rows)
	if err != nil {
		return processedRequest{}, err
	}

	return processedRequest{
		Account: acc,
		User:    user,
		Job:     job,
	}, nil
}

func createdAccMail(token string) string {
	return fmt.Sprintf(`
		<html>
		    <body>
			<!-- English version -->
			<p>Hello,</p>
			<p>Your account has been created. You can now log in using the credentials provided below. For security reasons, please change your password after logging in:</p>
			<p><a href="https://zigatdiploma.org/new-user?token=%s">Click here to log in</a></p>

			<hr/>

			<!-- Slovene version -->
			<p>Pozdravljeni,</p>
			<p>Vaš račun je bil ustvarjen. Zdaj se lahko prijavite z naslednjimi podatki. Zaradi varnosti prosimo, da po prijavi spremenite geslo:</p>
			<p><a href="https://zigatdiploma.org/new-user?token=%s">Kliknite tukaj za prijavo</a></p>

			<p>Regards / Lep pozdrav,<br/>The Diploma Project</p>
		    </body>
		</html>`, token, token)
}

func defaultPW(user *types.User) string {
	return fmt.Sprintf("%s%d%s", Capitalize(FirstNChars(user.FirstName, 2)), user.BirthDate.Year(), Capitalize(FirstNChars(user.LastName, 2)))
}

func InsertAccount(tx *sql.Tx, account types.NewAccount) (types.Account, error) {
	err := VerifyEmail(account.Email, "ZeroBounce1")
	if err != nil {
		return types.Account{}, err
	}

	acc := types.Account{
		UserID: account.UserID,
		Email:  account.Email,
	}

	userQuery := `
		INSERT INTO mobile.accounts
		(user_id, email, lang_code) VALUES
			($1, $2, $3)
		returning id`

	var aId int
	err = tx.QueryRow(
		userQuery,
		account.UserID,
		account.Email,
		account.LangCode,
	).Scan(&aId)
	if err != nil {
		log.Printf("Error insert account: %v\n", err)
		return types.Account{}, fmt.Errorf("Napaka pri vnosu računa v podatkovno bazo")
	}

	acc.ID = aId

	return acc, nil
}

func freshAccount(pRows processedRequest) (types.Account, error) {
	tx, err := MobileDB.Begin()
	if err != nil {
		return types.Account{}, fmt.Errorf("Failed to begin transaction")
	}
	defer tx.Rollback()

	userId, err := InsertUser(tx, pRows.User)
	if err != nil {
		return types.Account{}, err
	}

	pRows.User.ID = userId
	pRows.Account.UserID = userId

	acc, err := InsertAccount(tx, pRows.Account)
	if err != nil {
		return types.Account{}, err
	}

	pRows.Job.UserID = userId
	err = insertUserJob(tx, pRows.Job)
	if err != nil {
		return types.Account{}, err
	}

	err = tx.Commit()
	if err != nil {
		return types.Account{}, fmt.Errorf("Failed to commit transaction: %v", err)
	}

	return acc, nil
}

func ProcessBoardRequest(w http.ResponseWriter, r *http.Request) {
	var req types.BoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondIfErr(w, fmt.Errorf("failed to decode request body: %w", err), http.StatusBadRequest)
		return
	}

	pRows, err := processRequestRows(req.Rows)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	acc, err := freshAccount(pRows)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	rawToken, err := CreateToken(acc.ID, "first_login", 24*time.Hour)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	err = SendEmail(pRows.Account.Email, "Account Creation", createdAccMail(rawToken))
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getHashedPW(email string) (string, error) {
	query := "SELECT pw FROM mobile.accounts WHERE email = $1"
	var pw string

	err := MobileDB.QueryRow(query, email).Scan(&pw)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch acc's pw: %v", err)
	}

	return pw, nil
}

func CreateToken(accountID int, tokenType string, duration time.Duration) (string, error) {
	rawToken, err := GenerateSecureToken()
	if err != nil {
		return "", err
	}

	hashed := HashToken(rawToken)
	expiration := time.Now().Add(duration)

	token := types.HashedToken{
		AccountID: accountID,
		Token:     &hashed,
		Type:      tokenType,
		ExpiresAt: expiration,
	}

	if err := insertHashedToken(token); err != nil {
		return "", err
	}

	return rawToken, nil
}

func VerifyLogin(w http.ResponseWriter, r *http.Request) {
	var clientLogin types.ClientLogin
	err := json.NewDecoder(r.Body).Decode(&clientLogin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPw, err := getHashedPW(clientLogin.Email)
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(clientLogin.PW)); err != nil {
		log.Printf("invalid email or password: %v", err.Error())
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	acc, err := AccountByEmail(clientLogin.Email)
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}

	user, err := GetUser(acc.UserID)
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}

	secureToken, err := CreateToken(acc.ID, "secure_token", 7*24*time.Hour)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	repeatToken, err := CreateToken(acc.ID, "repeat_token", 30*24*time.Hour)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]any{
		"user":         user,
		"account":      acc,
		"secure_token": secureToken,
		"repeat_token": repeatToken,
	})
}

func AccountByEmail(email string) (types.Account, error) {
	query := `SELECT
			id,
			user_id,
			email,
			created_at,
			lang_code
		FROM mobile.accounts
		WHERE
			email = $1
	`
	var acc types.Account

	err := MobileDB.QueryRow(query, email).Scan(&acc.ID, &acc.UserID, &acc.Email, &acc.CreatedAt, &acc.LangCode)
	if err != nil {
		return types.Account{}, fmt.Errorf("Failed to fetch account by email: %v", err)
	}

	return acc, nil
}

func checkSetPWAuth(r *http.Request, req types.SetPW) (int, error) {
	aId, _ := headerAccId(r)

	if req.AccountID != nil && !AuthToken(r) {
		return -1, fmt.Errorf("account ids do not match")
	} else if req.Token != nil {
		token, err := MatchingToken(types.HashedTokenLookup{
			Type:  Pointer("first_login"),
			Token: req.Token,
		})
		if err != nil {
			return -1, fmt.Errorf("No matching token")
		}

		aId = token.AccountID
	}
	log.Printf("returning acc_id: %v", aId)

	return aId, nil
}

func SetPWReq(w http.ResponseWriter, r *http.Request) {
	var req types.SetPW
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	aId, err := checkSetPWAuth(r, req)
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}
	req.AccountID = &aId

	err = setPW(req)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	err = DeleteHashedTokens(types.HashedTokenLookup{AccountID: &aId})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	acc, err := AccountById(aId)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	user, err := GetUser(acc.UserID)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	secureToken, err := CreateToken(acc.ID, "secure_token", 7*24*time.Hour)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	repeatToken, err := CreateToken(acc.ID, "repeat_token", 30*24*time.Hour)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(map[string]any{
		"user":         user,
		"account":      acc,
		"secure_token": secureToken,
		"repeat_token": repeatToken,
	})
}

func setPW(setPW types.SetPW) error {
	hashedPW, err := bcrypt.GenerateFromPassword([]byte(setPW.RawPW), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash pw: %v", err)
	}

	query := "UPDATE mobile.accounts SET PW = $1 WHERE id = $2"
	_, err = MobileDB.Exec(query, hashedPW, setPW.AccountID)
	if err != nil {
		log.Printf("Error updating the password, %v", err)
		return fmt.Errorf("Failed to set new password: %v", err)
	}

	return nil
}

func AccountById(aId int) (types.Account, error) {
	query := `SELECT
			id,
			user_id,
			email,
			created_at,
			lang_code
		FROM mobile.accounts
		WHERE
			id = $1
	`
	var acc types.Account

	err := MobileDB.QueryRow(query, aId).Scan(&acc.ID, &acc.UserID, &acc.Email, &acc.CreatedAt, &acc.LangCode)
	if err != nil {
		return types.Account{}, fmt.Errorf("Failed to query for acc by id: %v", err)
	}

	return acc, nil
}

func AccLangCode(accId int) string {
	query := `
	SELECT
	    lang_code
	FROM mobile.accounts
	WHERE
	    id = $1`

	log.Printf("searching for langCode of %v", accId)

	var langCode string

	err := MobileDB.QueryRow(query, accId).Scan(&langCode)
	if err != nil {
		log.Printf("Failed to find acc's lang_code: %v\n", err)
		return ""
	}

	log.Printf("Lang code of %v is %v", accId, langCode)

	return langCode
}

func SetLangCode(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.AccLangReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	query := `UPDATE mobile.accounts SET lang_code = $2 WHERE id = $1`
	_, err := MobileDB.Exec(query, req.AccID, req.LangCode)
	if err != nil {
		log.Printf("Failed to language_code: %v", err)
		return
	}
}

func AccountIdByUserID(userId int) (int, error) {
	query := `SELECT id FROM mobile.accounts WHERE user_id = $1`

	var accId int
	err := MobileDB.QueryRow(query, userId).Scan(&accId)
	if err != nil {
		return -1, fmt.Errorf("Failed to query for acc_id by user_id: %v", err)
	}

	return accId, nil
}

func AccountByIdReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	aId, err := headerAccId(r)
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}

	accID, ok := getQueryParam[int](w, r, "account_id")
	if !ok {
		return
	}

	if aId != accID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	acc, err := AccountById(accID)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(acc)
}
