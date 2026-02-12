package slack

import (
	"backend/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func SlackInteraction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	payload := r.FormValue("payload")

	var base struct {
		Type string `json:"type"`
	}
	json.Unmarshal([]byte(payload), &base)
	fmt.Println(base.Type)

	switch base.Type {
	case "block_actions":
		w.WriteHeader(http.StatusOK)
		var interaction types.SlackInteractionPayload
		json.Unmarshal([]byte(payload), &interaction)

		switch interaction.Actions[0].ActionID {
		case "finished_obligation_upload":
			hideClickedButton(interaction)
			go finishedObligationUpload(interaction)

		case "finished_request_upload":
			hideClickedButton(interaction)
			go finishedRequestUpload(interaction)

		case "finished_document_upload":
			hideClickedButton(interaction)
			go finishedDocumentUpload(interaction)

		case "translate_button":
			translate(interaction)

		case "reject_action":
			go func(i types.SlackInteractionPayload) {
				hideClickedButton(interaction)
				var structuredValues types.StructuredModalValues
				if err := json.Unmarshal([]byte(i.Actions[0].Value), &structuredValues); err != nil {
					log.Println("Failed to unmarshal!")
					return
				}
				structuredValues.TriggerId = &i.TriggerID
				openModal(rejectionModal(structuredValues))
			}(interaction)

		case "accept_action":
			go func(i types.SlackInteractionPayload) {
				hideClickedButton(interaction)
				var structuredValues types.StructuredModalValues
				if err := json.Unmarshal([]byte(i.Actions[0].Value), &structuredValues); err != nil {
					log.Println("Failed to unmarshal!")
					return
				}
				structuredValues.TriggerId = &i.TriggerID
				openModal(acceptModal(structuredValues))
			}(interaction)

		case "update_button":
			listUserObligations(interaction)
		}

	case "view_submission":
		var interaction types.SlackInteractionPayload
		json.Unmarshal([]byte(payload), &interaction)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))

		go func(i types.SlackInteractionPayload) {
			switch interaction.View.CallbackID {
			case "create_obligation_modal":
				createObligationModalSubmission(i)

			case "message_user_modal":
				messageUserModalSubmission(i)

			case "upload_document_modal":
				uploadDocumentSubmission(i)

			case "notification_modal":
				notificationModalSubmission(i)

			case "rejection_modal":
				rejectionModalSubmission(i)

			case "accept_modal":
				acceptModalSubmission(i)

			case "remove_obligation_modal":
				removeObligationSubmission(i)

			case "assign_obligation_modal":
				assignObligationSubmission(i)

			case "update_user_modal":
				updateUserSubmission(i)

			case "set_group_modal":
				setGroupSubmission(i)

			case "employment_status_modal":
				updateEmploymentStatus(i)

			case "obligation_status_modal":
				updateObligationStatus(i)
			}
		}(interaction)

	case "block_suggestion":
		var interaction types.SlackSuggestionPayload
		json.Unmarshal([]byte(payload), &interaction)

		var response types.BlockSuggestionsResponse

		switch interaction.BlockID {
		case "user_select_block":
			userSelectSuggestions(interaction, &response)
		default:
			log.Printf("Suggestion block: '%s' not set!\n", interaction.BlockID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
