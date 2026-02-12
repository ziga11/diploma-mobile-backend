package slack

import (
	"backend/handlers"
	"backend/types"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"sync"
)

func addDocToObligation(fileId, slackUserID string, obligationId int) {
	if fileId != "" {
		return
	}

	filePath, mimetype, err := uploadSlackFileToDrive(fileId, slackUserID)
	if err != nil {
		log.Printf("Failed upload file to drive: %v", err)
		return
	}
	adminUserID := 1

	fileName := path.Base(filePath)

	doc, err := handlers.InsertDocument(types.Document{
		UserID:   adminUserID,
		DriveID:  filePath,
		Title:    fileName,
		Mimetype: &mimetype,
	})
	if err != nil {
		log.Printf("%v --> addDocToObligaiton", err)
		return
	}

	err = handlers.AddExampleDocToObligation(
		types.Oblg{
			ID:           obligationId,
			ExampleDocID: &doc.ID,
		})
	if err != nil {
		log.Printf("%v --> addDocToObligaiton", err)
		return
	}

}

func finishedObligationUpload(interaction types.SlackInteractionPayload) {
	userID := interaction.User.ID
	session, exists := obligationSessions[userID]
	if !exists {
		return
	}
	delete(obligationSessions, userID)

	addDocToObligation(session.FileID, session.SlackUserID, session.OTypes.Obligation.ID)

	_, err := sendMessage(types.SlackMessage{
		Channel:  interaction.User.ID,
		Text:     "",
		ThreadTS: &(interaction.Message.TS),
		Blocks:   fileUploadFinished("dolžnost"),
	})
	if err != nil {
		return
	}
}

func sendDocFCM(documentType string, userId int, doc types.Document) {
	accId, err := handlers.AccountIdByUserID(userId)
	if err != nil {
		log.Printf("failed to get accId by uID: %v", err)
		return
	}

	tokens, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID: &accId,
		Type:      handlers.Pointer("FCM"),
	})

	var wg sync.WaitGroup
	for _, token := range tokens {
		wg.Add(1)
		go func(dType string) {
			defer wg.Done()
			handlers.SendFCMNotification(types.SendNotification{
				UserId: userId,
				AccId:  accId,
				FCM:    token.Token,
				Translations: types.Translations{
					Si: types.LangContent{
						Title: fmt.Sprintf("Naložen je bil nov dokument tipa %s", dType),
						Body:  "Vaš dokument je bil uspešno prejet in shranjen.",
					},
					En: types.LangContent{
						Title: fmt.Sprintf("A new document of type %s has been uploaded", dType),
						Body:  "Your document has been successfully received and stored.",
					},
					Bs: types.LangContent{
						Title: fmt.Sprintf("Učitan je novi dokument tipa %s", dType),
						Body:  "Vaš dokument je uspješno zaprimljen i pohranjen.",
					},
				},
				Type:      "acnowledgement",
				Update:    "false",
				NavPage:   "/document-viewer",
				Arguments: fmt.Sprintf("%d", doc.ID),
			})
		}(documentType)
	}
	wg.Wait()

}

func finishedDocumentUpload(interaction types.SlackInteractionPayload) {
	userID := interaction.User.ID
	session, exists := documentSessions[userID]
	if !exists {
		return
	}
	delete(documentSessions, userID)

	if session.FileID == "" {
		log.Printf("No file attached... returning")
		return
	}

	filePath, mimetype, err := uploadSlackFileToDrive(session.FileID, session.SlackUserID)
	if err != nil {
		log.Printf("Failed upload file to drive: %v", err)
		return
	}

	doc, err := handlers.InsertDocument(types.Document{
		UserID:   session.UserID,
		DriveID:  filePath,
		Title:    session.Title,
		Mimetype: &mimetype,
		Type:     session.Type,
	})
	if err != nil {
		return
	}

	sendDocFCM(session.Type, session.UserID, doc)

	_, err = sendMessage(types.SlackMessage{
		Channel:  interaction.User.ID,
		Text:     "",
		ThreadTS: &(interaction.Message.TS),
		Blocks:   fileUploadFinished("dokument"),
	})
	if err != nil {
		return
	}

	delete(documentSessions, interaction.User.ID)
}

func uploadRequestFiles(fileIds []string, slackUID string, userId int) []int {
	docIds := []int{}

	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, fileID := range fileIds {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			filePath, mimetype, err := uploadSlackFileToDrive(id, slackUID)
			if err != nil {
				log.Printf("failed to upload file to drive: %v", err)
				return
			}
			fileName := path.Base(filePath)

			doc, err := handlers.InsertDocument(types.Document{
				UserID:   userId,
				DriveID:  filePath,
				Title:    fileName,
				Mimetype: &mimetype,
			})
			if err != nil {
				return
			}

			mu.Lock()
			docIds = append(docIds, doc.ID)
			mu.Unlock()
		}(fileID)
	}
	wg.Wait()

	return docIds
}

func finishedRequestUpload(interaction types.SlackInteractionPayload) {
	userID := interaction.User.ID
	session, exists := requestSessions[userID]
	if !exists {
		return
	}
	delete(requestSessions, userID)

	log.Printf("finished request uId --> %d", session.UserID)

	docIds := uploadRequestFiles(session.FileIDs, session.SlackUserID, session.UserID)

	senderId := 1
	sender, err := handlers.GetUser(senderId)
	if err != nil {
		return
	}
	recipient, err := handlers.GetUser(session.UserID)
	if err != nil {
		return
	}

	_, err = handlers.InsertEntireMessage(types.Message{
		Sender:       sender,
		Recipient:    recipient,
		ParentMsgID:  session.ParentMsgID,
		Translations: session.Translations,
		DocIds:       docIds,
	})
	if err != nil {
		return
	}

	accId, err := handlers.AccountIdByUserID(recipient.ID)
	if err != nil {
		log.Printf("Faield to fetch accId in finished request upload")
		return
	}

	rootMsg, err := handlers.FetchMessageRoot(*session.ParentMsgID)
	if err != nil {
		log.Printf("Failed to fetch message root: %v", err)
		return
	}

	handlers.SendFCMNotification(types.SendNotification{
		UserId:       session.UserID,
		AccId:        accId,
		FCM:          session.FCM,
		Translations: session.Translations,
		Type:         "acnowledgement",
		ThreadTS:     &session.ThreadTs,
		Update:       "false",
		NavPage:      "/message-page",
		Arguments:    fmt.Sprintf("%d", rootMsg.MessageID),
	})

	_, err = sendMessage(types.SlackMessage{
		Channel:  interaction.User.ID,
		Text:     "",
		ThreadTS: &interaction.Message.TS,
		Blocks:   fileUploadFinished("prošnja"),
	})
	if err != nil {
		return
	}
}

func updateBlocks(blocks []json.RawMessage, en []types.Translation, bs []types.Translation) []json.RawMessage {
	for i, blockData := range blocks {
		var inputBlock types.InputBlock
		if err := json.Unmarshal(blockData, &inputBlock); err != nil {
			continue
		}

		if inputBlock.Type != "input" {
			continue
		}

		baseID, hasV2 := strings.CutSuffix(inputBlock.BlockID, "_v2")
		newID := inputBlock.BlockID
		if hasV2 {
			newID = baseID
		} else {
			newID = inputBlock.BlockID + "_v2"
		}

		updated := false

		switch baseID {
		case "title_block_en":
			inputBlock.Element.InitialValue = en[0].Text
			updated = true
		case "body_block_en":
			inputBlock.Element.InitialValue = en[1].Text
			updated = true
		case "title_block_bs":
			inputBlock.Element.InitialValue = bs[0].Text
			updated = true
		case "body_block_bs":
			inputBlock.Element.InitialValue = bs[1].Text
			updated = true
		}

		if !updated {
			continue
		}

		inputBlock.BlockID = newID

		updatedBlockData, err := json.Marshal(inputBlock)
		if err != nil {
			log.Printf("Error marshaling updated block: %v", err)
			continue
		}
		blocks[i] = updatedBlockData
	}

	return blocks
}

func translate(interaction types.SlackInteractionPayload) {
	view := interaction.View
	modalValues := ExtractModalValues(view)

	enTr, err := handlers.Translate("SL", "EN", []string{modalValues.Translations.Si.Title, modalValues.Translations.Si.Body})
	if err != nil {
		log.Printf("Error translating --> %s", err)
	}

	bsTr, err := handlers.Translate("SL", "BS", []string{modalValues.Translations.Si.Title, modalValues.Translations.Si.Body})
	if err != nil {
		log.Printf("Error translating --> %s", err)
	}

	updatedBlocks := updateBlocks(view.Blocks, enTr, bsTr)

	payload := types.UpdatePayload{
		ViewID: view.ID,
		Hash:   view.Hash,
		View: types.UpdateView{
			CallbackID: view.CallbackID,
			Type:       view.Type,
			Title:      view.Title,
			Close:      view.Close,
			Submit:     view.Submit,
			Blocks:     updatedBlocks,
			Metadata:   view.Metadata,
		},
	}

	updateModal(payload)
}

func listUserObligations(interaction types.SlackInteractionPayload) {
	view := *interaction.View

	userIdStr := view.State.Values["user_select_block"]["user_select"].SelectedOption.Value

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		log.Printf("Failed to convert userId to int: %v", err)
		return
	}

	assignedObs, err := handlers.UserObligations(
		types.UserLang{
			UserID: userId,
		})
	if err != nil {
		return
	}
	obligationBlock := listUserObligationsBlock(assignedObs)

	rawObligationBlock, err := json.Marshal(obligationBlock)
	if err != nil {
		log.Printf("Error marshaling updated block: %v", err)
	}

	obligationStatuses := obligationStatuses()
	rawObligationStatuses, err := json.Marshal(obligationStatuses)
	if err != nil {
		log.Printf("Error marshaling updated block: %v", err)
	}

	view.Blocks = append(view.Blocks, rawObligationBlock, rawObligationStatuses)

	payload := types.UpdatePayload{
		ViewID: view.ID,
		Hash:   interaction.View.Hash,
		View: types.UpdateView{
			CallbackID: view.CallbackID,
			Type:       view.Type,
			Title:      view.Title,
			Close:      view.Close,
			Submit:     view.Submit,
			Blocks:     view.Blocks,
		},
	}

	updateModal(payload)
}
