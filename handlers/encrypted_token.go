package handlers

import (
	"backend/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lib/pq"
)

func DeleteEncryptedTokensReq(w http.ResponseWriter, r *http.Request) {
	var token types.EncryptedToken
	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
		return
	}

	var t []byte
	if token.Token != "" {
		t, _, err = getEncryptedToken(token.AccountID, token.Type, token.Token)
		if RespondIfErr(w, err, http.StatusInternalServerError) {
			return
		}
	}
	err = DeleteEncryptedTokens(types.EncryptedTokenLookup{
		ExpiredOnly: false,
		AccountID:   &token.AccountID,
		Type:        &token.Type,
		Token:       &t})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getEncryptedToken(accountID int, tokenType string, rawToken string) ([]byte, []byte, error) {
	query := `SELECT
			token,
			nonce
		  FROM mobile.encrypted_token
		  WHERE account_id = $1 AND
			type = $2 AND
			hashed_token = $3`

	var nonce []byte
	var enT []byte

	log.Printf("%v %v %v", accountID, tokenType, HashToken(rawToken))
	err := MobileDB.QueryRow(query, accountID, tokenType, HashToken(rawToken)).Scan(&enT, &nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to query encrypted token: %v", err)
	}

	return enT, nonce, nil
}

/*INFO: Passed token must be encrypted */
func DeleteEncryptedTokens(filter types.EncryptedTokenLookup) error {
	query := `DELETE FROM mobile.encrypted_token WHERE `
	args := []any{}
	i := 1

	if filter.ExpiredOnly {
		query += "expires_at < NOW()"
	} else {
		query += "expires_at >= NOW()"
	}

	if filter.AccountID == nil && !filter.ExpiredOnly {
		return fmt.Errorf("Cannot delete all tokens that are not expired and have no acc_id")
	}

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
		query += fmt.Sprintf(" AND token = $%d", i)
		args = append(args, *filter.Token)
		i++
	}

	_, err := MobileDB.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to delete token: %v", err)
		return fmt.Errorf("Failed to delete token: %v", err)
	}

	return nil
}

func FetchEncryptedTokens(filter types.EncryptedTokenLookup) ([]types.EncryptedToken, error) {
	query := `SELECT 
			account_id,
			token,
			nonce,
			type,
			expires_at
		  FROM mobile.encrypted_token
		  WHERE expires_at > NOW()`
	args := []any{}
	i := 1

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

	rows, err := MobileDB.Query(query, args...)
	var enTokens []types.EncryptedToken
	for rows.Next() {
		var t []byte
		var n []byte
		var enToken types.EncryptedToken
		rows.Scan(&enToken.AccountID, &t, &n, &enToken.Type, &enToken.ExpiresAt)

		dt, err := decryptAESGCM(t, n)
		if err != nil {
			return nil, fmt.Errorf("Failed to decrypt token: %v", err)
		}
		enToken.Token = dt

		enTokens = append(enTokens, enToken)
	}
	if err != nil {
		log.Printf("Failed to delete token: %v", err)
		return nil, fmt.Errorf("Failed to delete token: %v", err)
	}

	return enTokens, nil
}

func InsertEncryptedTokenReq(w http.ResponseWriter, r *http.Request) {
	var token types.EncryptedToken
	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = insertEncryptedToken(token)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func insertEncryptedToken(token types.EncryptedToken) error {
	query := `INSERT INTO mobile.encrypted_token(
		    account_id,
		    token,
		    hashed_token,
		    nonce,
		    type,
		    expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (hashed_token)
		DO UPDATE SET
		    account_id = EXCLUDED.account_id,
		    expires_at = EXCLUDED.expires_at,
		    token = EXCLUDED.token,
		    nonce = EXCLUDED.nonce;
		`

	if token.Token == "" {
		return fmt.Errorf("Failed to insert encrypted token, token empty")
	}

	log.Printf("saving token %s", token.Token)
	log.Printf("%+v", token)

	t, n, err := encryptAESGCM(token.Token)
	if err != nil {
		return fmt.Errorf("Failed to encrypt the token")
	}

	_, err = MobileDB.Exec(query, token.AccountID, t, HashToken(token.Token), n, token.Type, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("Failed to insert token: %w", err)
	}

	return nil
}

func FCMsOfgroups(ctx context.Context, listOfGroups []string) ([]types.EncryptedToken, error) {
	if len(listOfGroups) == 0 {
		return []types.EncryptedToken{}, nil
	}

	query := `
		SELECT  a.id,
			t.token,
			t.nonce
		FROM mobile.user_group_link ugl
		JOIN mobile.accounts a ON
			a.user_id = ugl.user_id
		JOIN mobile.groups g ON
			g.id = ugl.group_id
		JOIN mobile.encrypted_token t ON
			t.account_id = a.id AND
			t.type = 'FCM'
		WHERE
			g.id = ANY($1::int[]) AND t.token IS NOT NULL;
	`

	rows, err := MobileDB.QueryContext(ctx, query, pq.Array(listOfGroups))
	if err != nil {
		log.Printf("Query execution error: %v", err)
		return nil, fmt.Errorf("Failed to query token groups: %v", err)
	}
	defer rows.Close()

	var enTokens []types.EncryptedToken

	for rows.Next() {
		var enToken types.EncryptedToken
		var t []byte
		var n []byte
		if err := rows.Scan(&enToken.AccountID, &t, &n); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil, fmt.Errorf("Failed to scan row: %v", err)
		}

		log.Printf("%+v", enToken)

		dt, err := decryptAESGCM(t, n)
		if err != nil {
			log.Printf("Failed to decrypt token: %v\ndecrypted --> %v", err, dt)
			return nil, fmt.Errorf("Failed to decrypt token")
		}
		enToken.Token = dt

		enTokens = append(enTokens, enToken)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
	}

	log.Printf("encrypted tokens --> %+v", enTokens)

	return enTokens, nil
}
