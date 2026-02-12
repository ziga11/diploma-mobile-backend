package handlers

import (
	"backend/types"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

func FetchNotificationIdByContent(langContent types.LangContent) *int {
	query := `
		SELECT
		    notification_id
		FROM mobile.notification_translations
		WHERE
		    title = $1 AND
		    body = $2`

	var obligationId int
	err := MobileDB.QueryRow(query, langContent.Title, langContent.Body).Scan(&obligationId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Such notification does not exist")
		}
		return nil
	}

	return &obligationId
}

func InsertNotification(nType string) (int, error) {
	var nId int
	query := `INSERT INTO mobile.notification(type) VALUES ($1) returning id`

	err := MobileDB.QueryRow(query, nType).Scan(&nId)
	if err != nil {
		log.Printf("Failed to process notification: %v --> InsertNotification", err)
		return -1, fmt.Errorf("Failed to insert notification: %v", err)
	}

	return nId, nil
}

func InsertUserNotification(un types.UserNotification) (int, error) {
	query := `INSERT INTO mobile.user_notifications(
			notification_id,
		 	user_id,
			read,
			suitable,
			thread_ts)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING link_id`

	var linkId int

	err := MobileDB.QueryRow(query,
		un.NotificationID, un.UserID, un.Read, un.Suitable, un.ThreadTs).Scan(&linkId)
	if err != nil {
		log.Printf("Failed to process notification: %v --> insertUserNotification", err)
		return -1, fmt.Errorf("Failed to process notificaiton: %v", err)
	}

	return linkId, nil
}

func InsertNotificationTranslations(n types.NotificationTranslations) error {
	query := `INSERT INTO mobile.notification_translations(
			notification_id,
			language_code,
			title,
			body)
		VALUES ($1, $2, $3, $4)`

	stmt, err := MobileDB.Prepare(query)
	if err != nil {
		log.Printf("Failed to prepare query: %v\n", err)
		return err
	}
	defer stmt.Close()

	entries := []struct {
		Code string
		Data types.LangContent
	}{
		{"si", n.Translations.Si},
		{"en", n.Translations.En},
		{"bs", n.Translations.Bs},
	}

	for _, e := range entries {
		_, err := stmt.Exec(n.NotificationID, e.Code, e.Data.Title, e.Data.Body)
		if err != nil {
			log.Printf("Failed to insert notification translations: %v\n", err)
			return err
		}
	}

	return nil
}

func fetchUserNotifications(ctx context.Context, userId int) ([]types.Notification, error) {
	query := `
		SELECT 
		    link_id,
		    read,
		    date,
		    type,
		    suitable,
		    thread_ts,
		    translations
		FROM mobile.fetch_notifications($1)`

	rows, err := MobileDB.QueryContext(ctx, query, userId)
	if err != nil {
		log.Printf("Failed to query user notifications: %v", err)
		return nil, fmt.Errorf("Failed to query user notifications: %v", err)
	}
	defer rows.Close()

	notifications := []types.Notification{}
	for rows.Next() {
		var notification types.Notification
		if err := rows.Scan(
			&notification.LinkID,
			&notification.Read,
			&notification.Date,
			&notification.Type,
			&notification.Suitable,
			&notification.ThreadTs,
			&notification.Translations); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil, fmt.Errorf("Failed to scan user notifications row: %v", err)
		}

		notifications = append(notifications, notification)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
	}

	return notifications, nil
}

func FetchNotificationsReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}

	notifications, err := fetchUserNotifications(r.Context(), userId)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		http.Error(w, "Failed to encode documents: "+err.Error(), http.StatusInternalServerError)
	}
}

func SetReadNotification(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.UserNotification
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	query := `UPDATE mobile.user_notifications
			SET read = $1
		  WHERE
			user_id = $2 AND
			link_id = $3`
	_, err := MobileDB.Exec(query, true, req.UserID, req.LinkID)

	if err != nil {
		http.Error(w, "Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Update successful"))
}

func SetSuitableNotification(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.UserNotification
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	query := `UPDATE mobile.user_notifications
			SET suitable = $1
		  WHERE
			user_id = $2 AND
			link_id = $3`
	_, err := MobileDB.Exec(query, req.Suitable, req.UserID, req.LinkID)

	if err != nil {
		http.Error(w, "Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Update successful"))
}

func getFCMAccessToken(ctx context.Context) (string, error) {
	saJSON := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if saJSON == "" {
		return "", fmt.Errorf("FCM_SERVICE_ACCOUNT_JSON is not set")
	}

	b := []byte(saJSON)

	conf, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return "", fmt.Errorf("failed to create JWT config: %w", err)
	}

	token, err := conf.TokenSource(ctx).Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return token.AccessToken, nil
}

func sendFCMRequest(jsonMsg []byte, accessToken string) (*http.Response, error) {
	req, err := http.NewRequest(
		"POST",
		"https://fcm.googleapis.com/v1/projects/diploma-app-fb1ea/messages:send",
		bytes.NewBuffer(jsonMsg),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func handleFCMResponse(resp *http.Response) error {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode == http.StatusNotFound &&
		(strings.Contains(bodyStr, "UNREGISTERED") || strings.Contains(bodyStr, "InvalidRegistration")) {
		log.Printf("invalid device token: %s", bodyStr)
		return fmt.Errorf("invalid device token: %s", bodyStr)
	}

	log.Printf("notification failed: status=%d, body=%s", resp.StatusCode, bodyStr)
	return fmt.Errorf("notification failed: status=%d, body=%s", resp.StatusCode, bodyStr)
}

func InsertEntireNotification(sn types.SendNotification, userID int) (int, error) {
	nID, err := InsertNotification(sn.Type)
	if err != nil {
		log.Printf("Failed to insert notification")
		return -1, err
	}

	err = InsertNotificationTranslations(types.NotificationTranslations{
		Translations:   sn.Translations,
		NotificationID: nID,
	})
	if err != nil {
		log.Printf("Failed to insert notification translations")
		return -1, err
	}

	linkId, err := InsertUserNotification(types.UserNotification{
		UserID:         userID,
		NotificationID: nID,
		ThreadTs:       sn.ThreadTS,
	})
	if err != nil {
		log.Printf("Failed to insert user notification")
		return -1, err
	}

	return linkId, nil
}

func SendFCMNotification(notification types.SendNotification) error {
	ctx := context.Background()

	accessToken, err := getFCMAccessToken(ctx)
	if err != nil {
		return err
	}

	translationsJSON, err := json.Marshal(notification.Translations)
	if err != nil {
		log.Printf("Failed to marshal translations: %v", err)
		return err
	}

	linkId, err := InsertEntireNotification(notification, notification.UserId)
	if err != nil {
		return err
	}

	jsonMsg, err := json.Marshal(
		types.FCMMessageWrapper{
			Message: types.FCMMessage{
				Token: notification.FCM,
				Data: map[string]string{
					"type":         notification.Type,
					"link_id":      fmt.Sprintf("%d", linkId),
					"update":       notification.Update,
					"nav_page":     notification.NavPage,
					"arguments":    notification.Arguments,
					"translations": string(translationsJSON),
					"date":         time.Now().Format(time.RFC3339),
				},
			},
		})
	if err != nil {
		log.Printf("Failed to marshal fcm notification")
		return err
	}

	resp, err := sendFCMRequest(jsonMsg, accessToken)
	if err != nil {
		log.Printf("Failed to send fcm request: %v", err)
		return err
	}

	return handleFCMResponse(resp)
}
