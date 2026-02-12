package handlers

import (
	"backend/types"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func insertHashedToken(token types.HashedToken) error {
	query := `INSERT INTO mobile.hashed_token(
		    account_id,
		    token,
		    type,
		    expires_at
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		`

	_, err := MobileDB.Exec(query, token.AccountID, token.Token, token.Type, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("Failed to insert token: %v", err)
	}

	return nil
}

func ResetPwTokenReq(w http.ResponseWriter, r *http.Request) {
	var token types.HashedToken
	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
		return
	}

	t, err, code := passwordToken(token)
	if RespondIfErr(w, err, code) {
		return
	}

	resetURL := fmt.Sprintf("https://zigatdiploma.org/reset-password?token=%s&email=%s", t, *token.Email)

	emailBody := fmt.Sprintf(`
	<html>
	    <body>
		<p>Hello,</p>
		<p>You requested a password reset. Click the link below to set a new password:</p>
		<p><a href="%[1]s">Reset Password</a></p>

		<hr/>

		<p>Pozdravljeni,</p>
		<p>Prejeli smo zahtevo za ponastavitev vašega gesla. Kliknite na spodnjo povezavo za nastavitev novega gesla:</p>
		<p><a href="%[1]s">Ponastavi geslo</a></p>

		<br/>
		<p>Regards / Lep pozdrav,<br/>The Diploma Project</p>
	    </body>
	</html>`, resetURL)

	err = SendEmail(*token.Email, "Password Reset Request", emailBody)
	if RespondIfErr(w, err, code) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CountTokens(filter types.HashedTokenLookup) (int, error) {
	if filter.AccountID == nil || filter.Type == nil {
		return -1, fmt.Errorf("no accountId or type provided")
	}

	var count int
	err := MobileDB.QueryRow(`
		SELECT COUNT(*) 
		FROM mobile.hashed_token
		WHERE account_id = $1 AND
		      type = $2 AND
		      expires_at > NOW()`,
		filter.AccountID, filter.Type).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func MatchingToken(token types.HashedTokenLookup) (types.HashedToken, error) {
	if token.Token == nil || token.Type == nil {
		return types.HashedToken{}, fmt.Errorf("token or type is not provided")
	}

	var t types.HashedToken
	err := MobileDB.QueryRow(`
		SELECT account_id, type, expires_at
		FROM mobile.hashed_token
		WHERE token = $1
		  AND type = $2
		  AND expires_at > NOW()
	`, HashToken(*token.Token), token.Type).Scan(
		&t.AccountID,
		&t.Type,
		&t.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return types.HashedToken{}, fmt.Errorf("invalid or expired token")
		}
		return types.HashedToken{}, err
	}

	return t, nil
}

func MatchingTokenReq(w http.ResponseWriter, r *http.Request) {
	accID, ok := getQueryParam[int](w, r, "account_id")
	if !ok {
		return
	}
	rawToken, ok := getQueryParam[string](w, r, "token")
	if !ok {
		return
	}
	tType, ok := getQueryParam[string](w, r, "type")
	if !ok {
		return
	}

	t, err := MatchingToken(types.HashedTokenLookup{Token: &rawToken, Type: &tType})
	if RespondIfErr(w, err, http.StatusUnauthorized) {
		return
	}

	if t.AccountID != accID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteHashedTokensReq(w http.ResponseWriter, r *http.Request) {
	var token types.HashedToken
	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = DeleteHashedTokens(types.HashedTokenLookup{AccountID: &token.AccountID, Type: &token.Type, Token: token.Token})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteHashedTokens(filter types.HashedTokenLookup) error {
	query := `DELETE FROM mobile.hashed_token WHERE expires_at < NOW()`
	args := []any{}
	i := 1

	if filter.AccountID != nil || filter.Type != nil || filter.Token != nil {
		query += " OR (1=1"
		if filter.AccountID != nil {
			query += fmt.Sprintf(" AND account_id = $%d", i)
			args = append(args, *filter.AccountID)
			i++
		}
		if filter.Type != nil {
			query += fmt.Sprintf(" AND type = $%d", i)
			args = append(args, *filter.Type)
			i++
		}
		if filter.Token != nil {
			hashed := HashToken(*filter.Token)
			query += fmt.Sprintf(" AND token = $%d", i)
			args = append(args, hashed)
			i++
		}
		query += ")"
	}

	_, err := MobileDB.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to delete token: %v", err)
		return fmt.Errorf("Failed to delete token: %v", err)
	}

	return nil
}

func passwordToken(token types.HashedToken) (string, error, int) {
	acc, err := AccountByEmail(*token.Email)
	if err != nil {
		return "", nil, http.StatusOK
	}

	tCount, err := CountTokens(types.HashedTokenLookup{AccountID: &acc.ID, Type: Pointer("forgottenPW")})
	if err != nil {
		return "", fmt.Errorf("failed fetching tokens"), http.StatusInternalServerError
	}

	if tCount >= 3 {
		return "", fmt.Errorf("You have reached the maximum number of password reset requests for today."), http.StatusForbidden
	}

	t, err := CreateToken(acc.ID, "forgottenPW", 30*time.Minute)
	if err != nil {
		return "", err, http.StatusInternalServerError
	}

	return t, nil, http.StatusOK
}

func ResetPW(w http.ResponseWriter, r *http.Request) {
	var req types.ResetPW
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	acc, err := AccountByEmail(req.Email)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	tCount, err := CountTokens(types.HashedTokenLookup{AccountID: &acc.ID, Type: Pointer("forgottenPW")})
	if RespondIfErr(w, err, http.StatusNotFound) {
		return
	}

	if tCount == 0 {
		RespondIfErr(w, fmt.Errorf("No token in the DB"), http.StatusNotFound)
		return
	}

	err = setPW(types.SetPW{RawPW: req.PW, AccountID: &acc.ID})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	err = DeleteHashedTokens(types.HashedTokenLookup{Type: Pointer("forgottenPW"), AccountID: &acc.ID})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}
