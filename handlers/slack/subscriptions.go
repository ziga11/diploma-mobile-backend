package slack

import (
	"backend/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func messageEvent(evt types.SlackEvent, ie types.InnerEvent) {
	var msgEvent types.MessageEvent
	if err := json.Unmarshal(evt.Event, &msgEvent); err != nil {
		log.Println("failed to parse message event:", err)
		return
	}

	if msgEvent.Subtype != "file_share" {
		return
	}

	var entityType string

	if session, ok := obligationSessions[ie.UserID]; ok {
		session.FileID = msgEvent.Files[0].ID
		obligationSessions[ie.UserID] = session

		entityType = "obligation"
	} else if session, ok := requestSessions[ie.UserID]; ok {
		for _, file := range msgEvent.Files {
			session.FileIDs = append(session.FileIDs, file.ID)
		}

		requestSessions[ie.UserID] = session

		entityType = "request"
	} else if session, ok := documentSessions[ie.UserID]; ok {
		session.FileID = msgEvent.Files[0].ID
		documentSessions[ie.UserID] = session

		entityType = "document"
	}

	messageText := fmt.Sprintf(
		`📎 Datoteka '%s' je pripravljena za shranjevanje!`, msgEvent.Files[0].Name,
	)

	if entityType != "request" {
		messageText += "*Opomba: Shranite lahko samo eno datoteko. Kliknite Dokončano za potrditev.*\n" +
			"*Če naložite novo datoteko, bo ta prepisala obstoječo.*"
	}

	sendMessage(types.SlackMessage{
		Channel:  ie.UserID,
		Text:     "",
		ThreadTS: &ie.ThreadTs,
		Blocks:   savedFileMessage(messageText, entityType),
	})
}

func SlackSubscription(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	var evt types.SlackEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch evt.Type {
	case "url_verification":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, evt.Challenge)
		return
	case "event_callback":
		var ie types.InnerEvent
		if err := json.Unmarshal(evt.Event, &ie); err != nil {
			log.Println("failed to parse inner event:", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		switch ie.Type {
		case "message":
			messageEvent(evt, ie)
		default:
			log.Printf("Unhandled event type: %s\n", ie.Type)
		}
	}
	w.WriteHeader(http.StatusOK)
}
