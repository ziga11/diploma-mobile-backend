package slack

import (
	"backend/handlers"
	"context"
	"fmt"
	"net/http"
)

func SlackCommandResponse(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	for key, values := range r.Form {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	triggerID := r.FormValue("trigger_id")
	slackUserID := r.FormValue("user_id")
	cmdText := r.FormValue("text")
	cmd := r.FormValue("command")

	ctx := context.Background()

	w.WriteHeader(http.StatusOK)

	fmt.Printf("Slash command received from %s: %s\n", slackUserID, cmdText)
	go func() {
		switch cmd {
		case "/ustvari_dolznost":
			countryList := handlers.ListOfCountries(ctx)
			obligationCountries := countryList[:6]

			jobList := handlers.JobList(ctx)

			openModal(createObligationModal(jobList, obligationCountries, slackUserID, triggerID))

		case "/dodeli_dolznost":
			distinctObligations := handlers.DistinctObligations(ctx)
			openModal(assignObligationModal(triggerID, distinctObligations))

		case "/nalozi_dokument":
			openModal(uploadDocumentModal(triggerID, slackUserID))

		case "/izbrisi_dolznost":
			distinctObligations := handlers.DistinctObligations(ctx)
			openModal(removeObligationModal(triggerID, distinctObligations))

		case "/sporoci":
			openModal(messageUserModal(triggerID))

		case "/obvesti":
			groupList := handlers.ListOfGroups(ctx)
			openModal(notificationModal(groupList, triggerID))

		case "/doloci_skupino":
			groupList := handlers.ListOfGroups(ctx)
			openModal(setUserGroupModal(groupList, triggerID))

		case "/sprememba_podatkov":
			openModal(updateUserDataModal(triggerID))

		case "/doloci_status_zaposlitve":
			openModal(employmentStatusModal(triggerID))

		case "/doloci_status_dolznosti":
			openModal(setObligationStatusModal(triggerID))
		}
	}()
}
