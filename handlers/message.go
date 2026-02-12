package handlers

import (
	"backend/types"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func fetchMessageDocIds(mId int) ([]int, error) {
	query := `SELECT
			document_id
		  FROM mobile.message_document_link
		  WHERE message_id = $1`
	var docIds []int

	rows, err := MobileDB.Query(query, mId)
	if err != nil {
		log.Printf("Failed to select docIds where msgId = %v: %v", mId, err)
		return nil, fmt.Errorf("Failed to query docIds: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docId int
		if err := rows.Scan(&docId); err != nil {
			log.Printf("Failed to select docId row where msgId = %v: %v", mId, err)
			return nil, fmt.Errorf("Failed to scan docId row: %v", err)
		}

		docIds = append(docIds, docId)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
	}

	return docIds, nil
}

func FetchMessagesReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}
	msgId, err := QueryGetType[int](r, "message_id")

	msgs, err := FetchMessages(types.UserMessage{
		UserID:    userId,
		MessageID: TernaryOperator(err == nil, &msgId, nil),
	})

	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func FetchMessages(mReq types.UserMessage) ([]types.Message, error) {
	query := `
		SELECT 
		    message_id,
		    sender_id,
		    recipient_id,
		    date,
		    read,
		    translations
		FROM mobile.fetch_messages($1, $2)
	`

	var msgs []types.Message

	rows, err := MobileDB.Query(query, mReq.UserID, mReq.MessageID)
	if err != nil {
		log.Printf("Querying for messages failed: %v", err.Error())
		return nil, fmt.Errorf("Querying for messages failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg types.Message
		var senderId int
		var recipientId int
		if err := rows.Scan(&msg.ID, &senderId, &recipientId, &msg.Date, &msg.Read, &msg.Translations); err != nil {
			log.Printf("Failed to scan row: %v", err.Error())
			return nil, fmt.Errorf("Failed to scan message row: %v", err)
		}

		sender, err := GetUser(senderId)
		if err != nil {
			log.Printf("Failed to retrieve msg sender uId = %v: %v", senderId, err)
			return nil, err
		}
		recipient, err := GetUser(recipientId)
		if err != nil {
			log.Printf("Failed to retrieve msg recipient uId = %v: %v", recipientId, err)
			return nil, err
		}

		msg.Sender = sender
		msg.Recipient = recipient

		docIds, err := fetchMessageDocIds(msg.ID)
		if err != nil {
			return nil, err
		}

		msg.DocIds = docIds
		msgs = append(msgs, msg)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
	}

	return msgs, nil
}

func SetReadStatusReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req types.UserMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := SetReadStatus(req)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func SetReadStatus(req types.UserMessage) error {
	query := `UPDATE mobile.user_messages
			SET read = $1
		  WHERE
			message_id = $2 AND
			user_id = $3`

	_, err := MobileDB.Exec(query, req.Read, req.MessageID, req.UserID)

	if err != nil {
		return fmt.Errorf("Failed to update read status to Message: %v", err)
	}

	return nil
}

func MessageThreadReq(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, ok := getQueryParam[int](w, r, "user_id")
	if !ok {
		return
	}
	mId, ok := getQueryParam[int](w, r, "message_id")
	if !ok {
		return
	}

	msgs, err := fetchMessageThread(types.UserMessage{UserID: userId, MessageID: &mId})
	if RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(msgs); err != nil {
		http.Error(w, "Failed to encode messages: "+err.Error(), http.StatusInternalServerError)
	}
}

func fetchMessageThread(msgReq types.UserMessage) ([]types.Message, error) {
	msgs := []types.Message{}

	rows, err := MobileDB.Query(`
				SELECT message_id,
					sender_id,
					recipient_id,
					date,
					parent_msg_id,
					read,
					translations
				FROM mobile.fetch_message_thread($1::int, $2::int)`,
		msgReq.MessageID, msgReq.UserID)
	if err != nil {
		log.Printf("Failed to query database: %v\n", err.Error())
		return nil, fmt.Errorf("Failed to query db message_thread: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var msg types.Message
		var senderId int
		var recipientId int

		if err := rows.Scan(
			&msg.ID, &senderId, &recipientId, &msg.Date, &msg.ParentMsgID, &msg.Read, &msg.Translations); err != nil {
			log.Printf("Failed to read row: %v --> %v", err.Error(), http.StatusInternalServerError)
			return nil, fmt.Errorf("Failed to read message thread row: %v", err.Error())
		}

		sender, err := GetUser(senderId)
		if err != nil {
			return nil, err
		}
		recipient, err := GetUser(recipientId)
		if err != nil {
			return nil, err
		}

		msg.Sender = sender
		msg.Recipient = recipient

		docIds, err := fetchMessageDocIds(msg.ID)
		if err != nil {
			return nil, err
		}

		msg.DocIds = docIds
		msgs = append(msgs, msg)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil, fmt.Errorf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
	}

	return msgs, err
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	if !AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var msg types.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgId, err := InsertEntireMessage(msg)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(map[string]any{"message_id": msgId}); err != nil {
		http.Error(w, "Failed to encode messages: "+err.Error(), http.StatusInternalServerError)
	}
}

func FetchMessageRoot(msgId int) (types.MessageTranslations, error) {
	query := `SELECT * FROM mobile.fetch_message_root($1)`

	var msg types.MessageTranslations

	err := MobileDB.QueryRow(query, msgId).Scan(
		&msg.MessageID,
		&msg.Translations,
	)
	if err != nil {
		return types.MessageTranslations{}, fmt.Errorf("Failed to fetch message root of %d --> %v", msgId, err)
	}

	return msg, nil
}

func InsertEntireMessage(msg types.Message) (int, error) {
	tx, err := MobileDB.Begin()
	if err != nil {
		log.Println("Failed to start transaction:", err)
		return -1, fmt.Errorf("Failed to start transaction")
	}
	defer tx.Rollback()

	messageId, err := InsertMessage(tx, types.Msg{
		Sender:   msg.Sender,
		ParentId: msg.ParentMsgID,
		ThreadTS: msg.ThreadTS,
	})
	if err != nil {
		return -1, err
	}

	err = InsertMessageTranslations(tx, types.MessageTranslations{MessageID: messageId, Translations: msg.Translations})
	if err != nil {
		return -1, err
	}

	err = InsertUserMessages(tx, types.UserMessage{UserID: msg.Sender.ID, MessageID: &messageId, Read: true})
	if err != nil {
		return -1, err
	}

	err = InsertUserMessages(tx, types.UserMessage{UserID: msg.Recipient.ID, MessageID: &messageId, Read: false})
	if err != nil {
		return -1, err
	}

	for _, docId := range msg.DocIds {
		err = InsertMessageDocs(tx, types.MessageDocuments{MessageID: messageId, DocId: docId})
		if err != nil {
			return -1, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return -1, fmt.Errorf("Failed to commit transaction: %v", err)
	}

	return messageId, nil
}

func InsertMessage(tx *sql.Tx, msg types.Msg) (int, error) {
	var messageID int
	query := `INSERT INTO
		     mobile.message(creator_id, thread_ts, parent_msg_id) VALUES
		     ($1, COALESCE($2, ''), $3)
		  RETURNING id`

	err := MobileDB.QueryRow(query, msg.Sender.ID, msg.ThreadTS, msg.ParentId).Scan(&messageID)
	if err != nil {
		log.Printf("Failed to insert message: "+err.Error(), http.StatusInternalServerError)
		return -1, fmt.Errorf("Failed to insert message: %v", err.Error())
	}

	return messageID, nil
}

func InsertMessageTranslations(tx *sql.Tx, msg types.MessageTranslations) error {
	query := `INSERT INTO
			mobile.message_translations(message_id, title, body, language_code) VALUES
			($1, $2, $3, $4)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		log.Printf("Failed to prepare query: %v\n", err)
		return err
	}
	defer stmt.Close()

	entries := []struct {
		Code string
		Data types.LangContent
	}{
		{"si", msg.Translations.Si},
		{"en", msg.Translations.En},
		{"bs", msg.Translations.Bs},
	}

	for _, entry := range entries {
		_, err = stmt.Exec(msg.MessageID, entry.Data.Title, entry.Data.Body, entry.Code)
		if err != nil {
			log.Printf("Error inserting msg translations: %v\n", err)
			return fmt.Errorf("Failed to insert message translations: %v", err.Error())
		}
	}

	return nil
}

func InsertUserMessages(tx *sql.Tx, msg types.UserMessage) error {
	query := `INSERT INTO
			mobile.user_messages(user_id, message_id, read) VALUES
			($1, $2, $3)`

	_, err := tx.Exec(query, msg.UserID, msg.MessageID, msg.Read)
	if err != nil {
		log.Printf("Failed to insert user message link: "+err.Error(), http.StatusInternalServerError)
		return fmt.Errorf("Failed to insert user message: %v", err.Error())
	}

	return nil
}

func InsertMessageDocs(tx *sql.Tx, msg types.MessageDocuments) error {
	query := `INSERT INTO
			mobile.message_document_link(message_id, document_id) VALUES
			($1, $2)`

	_, err := tx.Exec(query, msg.MessageID, msg.DocId)
	if err != nil {
		log.Printf("Failed to link document: "+err.Error(), http.StatusInternalServerError)
		return fmt.Errorf("Failed to insert message documents: %v", err.Error())
	}

	return nil
}
