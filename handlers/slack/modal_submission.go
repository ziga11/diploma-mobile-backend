package slack

import (
	"backend/handlers"
	"backend/types"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func obligationTypesFromMValues(modalValues *types.ModalValues) (obligationTypes, error) {
	var oTypes obligationTypes

	allCountriesSize := 5
	workpermitStatusesSize := 5

	if len(modalValues.SelectedWorkpermits) == workpermitStatusesSize {
		oTypes.Applicability.WorkpermitStatus = []string{}
	}

	if len(modalValues.SelectedCountries) == allCountriesSize {
		oTypes.Applicability.CountryIDs = []int{}
	}

	oTypes.Translations.Translations = modalValues.Translations
	oTypes.Obligation.Uploadable = modalValues.Uploadable
	oTypes.Obligation.TextField = modalValues.TextField

	for _, job := range modalValues.SelectedJobs {
		companyId, err := strconv.Atoi(job)
		if err != nil {
			log.Println("Failed to convert job Id to int")
			return obligationTypes{}, fmt.Errorf("Failed to convert company Id to int: %v", err)
		}

		if oTypes.Applicability.JobIDs == nil {
			oTypes.Applicability.JobIDs = []int{}
		}

		oTypes.Applicability.JobIDs = append(oTypes.Applicability.JobIDs, companyId)
	}

	for _, country := range modalValues.SelectedCountries {
		countryId, err := strconv.Atoi(country)
		if err != nil {
			log.Println("Failed to convert country Id to int")
			return obligationTypes{}, fmt.Errorf("Failed to convert country Id to int: %v", err)
		}
		if oTypes.Applicability.CountryIDs == nil {
			oTypes.Applicability.CountryIDs = []int{}
		}

		oTypes.Applicability.CountryIDs = append(oTypes.Applicability.CountryIDs, countryId)
	}

	return oTypes, nil
}

func createObligationModalSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	oTypes, err := obligationTypesFromMValues(&modalValues)
	if err != nil {
		return
	}

	err = handlers.CreateEntireObligation(oTypes.Obligation, oTypes.Translations, oTypes.Applicability)
	if err != nil {
		return
	}

	_, err = sendMessage(types.SlackMessage{
		Channel: modalValues.StructuredMetadata.SlackUserId,
		Text:    "✅ Your obligation creation request has been created.\n\n📎 Please upload any related files here.",
		Blocks:  entityCreatedMessage("obligation"),
	})
	if err != nil {
		return
	}

	obligationSessions[interaction.User.ID] = obligationUploadSession{
		Channel:     interaction.Channel.ID,
		ThreadTs:    interaction.Message.TS,
		SlackUserID: interaction.User.ID,
		OTypes:      oTypes,
	}
}

func notificationModalSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)
	var selectedGroups []string

	for _, option := range modalValues.SelectedGroups {
		cleanGroupName := strings.ReplaceAll(option, "_", " ")
		selectedGroups = append(selectedGroups, cleanGroupName)
	}

	if len(selectedGroups) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenDataArr, err := handlers.FCMsOfgroups(ctx, selectedGroups)
	if err != nil {
		return
	}

	for _, tokenData := range tokenDataArr {
		acc, err := handlers.AccountById(tokenData.AccountID)
		if err != nil {
			log.Printf("Failed to send group notification: %v", err)
			return
		}

		err = handlers.SendFCMNotification(
			types.SendNotification{
				AccId:        acc.ID,
				UserId:       acc.UserID,
				FCM:          tokenData.Token,
				Translations: modalValues.Translations,
				Type:         modalValues.MessageType,
				ThreadTS:     interaction.Message.ThreadTS,
				Update:       "false",
				NavPage:      "/notification-page",
				Arguments:    "",
			})
		if err != nil {
			log.Printf("Failed to send notification to %v with token %v, %v", tokenData.AccountID, tokenData.Token, err)
		}
	}
}

func messageUserModalSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)
	userIdInt, err := strconv.Atoi(modalValues.UserID)
	if err != nil {
		log.Printf("Failed to convert userId to int\n")
		return
	}

	accId, err := handlers.AccountIdByUserID(userIdInt)
	if err != nil {
		log.Printf("error fetching acc by id")
		return
	}

	tokens, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{AccountID: &accId, Type: handlers.Pointer("FCM")})
	if err != nil {
		log.Printf("error fetching encrypted tokens")
		return
	}

	for _, token := range tokens {
		handlers.SendFCMNotification(
			types.SendNotification{
				AccId:        accId,
				UserId:       userIdInt,
				FCM:          token.Token,
				Translations: modalValues.Translations,
				Type:         modalValues.MessageType,
				ThreadTS:     interaction.Message.ThreadTS,
				Update:       "false",
				NavPage:      "/notification-page",
				Arguments:    "",
			})
	}
}

func rejectionModalSubmission(interaction types.SlackInteractionPayload) error {
	modalValues := ExtractModalValues(interaction.View)
	recipientId := modalValues.StructuredMetadata.UserId

	accId, err := handlers.AccountIdByUserID(recipientId)
	if err != nil {
		return fmt.Errorf("failed to get account ID for user %d: %w", recipientId, err)
	}

	threadTS := getThreadTS(&interaction.Message)

	if modalValues.StructuredMetadata.Channel == "obligation" {
		return handleRejectedObligation(modalValues, accId, recipientId, threadTS)
	}

	return handleRejectedMessage(modalValues, recipientId, accId, &interaction.Message.TS, threadTS)
}

func handleRejectedObligation(modalValues types.ModalValues, accId, userId int, threadTS *string) error {
	if modalValues.StructuredMetadata.ObligationId == nil {
		return fmt.Errorf("obligation ID is nil")
	}

	_, err := handlers.SetUserObligationStatus(&types.UserObligation{
		UserId:       userId,
		ObligationID: *modalValues.StructuredMetadata.ObligationId,
		Status:       "Incomplete",
	})
	if err != nil {
		return fmt.Errorf("failed to set obligation status: %w", err)
	}

	if err := handlers.RemoveUserObligationDocs(userId, *modalValues.StructuredMetadata.ObligationId); err != nil {
		log.Printf("Warning: failed to remove obligation docs: %v", err)
	}

	return sendAcknowledgementNotification(accId, userId, modalValues, threadTS)
}

func handleRejectedMessage(modalValues types.ModalValues, recipientId, accId int, messageTS, threadTS *string) error {
	const systemSenderId = 1

	sender, err := handlers.GetUser(systemSenderId)
	if err != nil {
		return fmt.Errorf("failed to get sender user: %w", err)
	}

	recipient, err := handlers.GetUser(recipientId)
	if err != nil {
		return fmt.Errorf("failed to get recipient user: %w", err)
	}

	_, err = handlers.InsertEntireMessage(types.Message{
		Sender:       sender,
		Recipient:    recipient,
		ParentMsgID:  modalValues.StructuredMetadata.MessageId,
		ThreadTS:     messageTS,
		Translations: modalValues.Translations,
	})
	if err != nil {
		return fmt.Errorf("failed to insert rejection message: %w", err)
	}

	return sendAcknowledgementNotification(accId, recipientId, modalValues, threadTS)
}

func sendAcknowledgementNotification(accId, userId int, modalValues types.ModalValues, threadTS *string) error {
	err := handlers.SendFCMNotification(types.SendNotification{
		AccId:        accId,
		UserId:       userId,
		FCM:          modalValues.StructuredMetadata.FcmToken,
		Translations: modalValues.Translations,
		Type:         "acknowledgement",
		ThreadTS:     threadTS,
		Update:       "false",
		NavPage:      "/notification-page",
		Arguments:    "",
	})
	if err != nil {
		return fmt.Errorf("failed to send acknowledgement notification: %w", err)
	}
	return nil
}

func getThreadTS(msg *types.SlackMessageMeta) *string {
	if msg == nil {
		return nil
	}
	return msg.ThreadTS
}

func acceptedObligation(modalValues types.ModalValues, threadTs *string) {
	userId := modalValues.StructuredMetadata.UserId
	obligationId := *modalValues.StructuredMetadata.ObligationId

	_, err := handlers.SetUserObligationStatus(&types.UserObligation{
		UserId:       userId,
		ObligationID: obligationId,
		Status:       "Completed",
	})
	if err != nil {
		log.Printf("Failed to set obligation status: %v", err)
		return
	}

	update := "false"

	if handlers.CheckCompletionStatus(userId) {
		err := handlers.AllObligationCompleted(userId)
		if err != nil {
			log.Printf("Failed to set new obligations: %v", err)
		}

		update = "true"
	}

	accId, err := handlers.AccountIdByUserID(userId)
	if err != nil {
		log.Printf("Failed to get accId by userID")
		return
	}

	err = handlers.SendFCMNotification(
		types.SendNotification{
			AccId:        accId,
			UserId:       userId,
			FCM:          modalValues.StructuredMetadata.FcmToken,
			Translations: modalValues.Translations,
			Type:         "acknowledgement",
			ThreadTS:     threadTs,
			Update:       update,
			NavPage:      "/list-obligations",
			Arguments:    fmt.Sprintf("%d", obligationId),
		})
	if err != nil {
		log.Printf("Failed to send accepted obligation notification")
	}
}

func acceptedRequest(modalValues types.ModalValues, interaction types.SlackInteractionPayload) {
	_, err := sendMessage(types.SlackMessage{
		Channel: interaction.User.ID,
		Text:    "✅ Your request has been created.\n\n📎 Please upload any related files here.",
		Blocks:  entityCreatedMessage("request"),
	})
	if err != nil {
		log.Printf("error occurred sending slack message: %v", err)
		return
	}

	log.Printf("accepted Request --> %v", modalValues)

	requestSessions[interaction.User.ID] = requestUploadSession{
		Channel:      interaction.Channel.ID,
		ThreadTs:     interaction.Message.TS,
		SlackUserID:  interaction.User.ID,
		UserID:       modalValues.StructuredMetadata.UserId,
		Translations: modalValues.Translations,
		FCM:          modalValues.StructuredMetadata.FcmToken,
		ParentMsgID:  modalValues.StructuredMetadata.MessageId,
	}
}

func acceptModalSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	log.Printf("%+v", modalValues)
	log.Printf("%+v", modalValues)

	if modalValues.StructuredMetadata.Channel == "obligation" {
		acceptedObligation(modalValues, interaction.Message.ThreadTS)
	} else {
		acceptedRequest(modalValues, interaction)
	}
}

func assignObligationSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)
	userId, err := strconv.Atoi(modalValues.UserID)
	if err != nil {
		log.Println("Failed to parse userID from multiselect")
		return
	}

	var obString string

	for _, val := range modalValues.Obligations {
		obligationId, err := strconv.Atoi(val)
		if err != nil {
			log.Println("Failed to parse obligationId from multiselect")
			return
		}

		if len(obString) != 0 {
			obString = fmt.Sprintf("%s, %d", obString, obligationId)
		} else {
			obString = fmt.Sprintf("%d", obligationId)
		}

		handlers.AssignSingleObligation(types.UserObligation{ObligationID: obligationId, UserId: userId})
	}

	accId, err := handlers.AccountIdByUserID(userId)
	if err != nil {
		log.Printf("error getting accId by userID: %v", err)
	}

	fcm, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID:   &accId,
		Type:        handlers.Pointer("FCM"),
		ExpiredOnly: false,
	})

	for _, token := range fcm {
		handlers.SendFCMNotification(types.SendNotification{
			AccId:  accId,
			UserId: userId,
			FCM:    token.Token,
			Translations: types.Translations{
				Si: types.LangContent{
					Title: "Dodeljena obveznost",
					Body:  "Dodeljena Vam je bila nova obveznost",
				},
				En: types.LangContent{
					Title: "New obligation",
					Body:  "You have been assigned a new obligation",
				},
				Bs: types.LangContent{
					Title: "Dodijeljena nova obaveza",
					Body:  "Dodijeljena vam je nova obaveza",
				},
			},
			Type:      "acknowledgement",
			Update:    "false",
			NavPage:   "/list-obligations",
			Arguments: obString,
		})
	}
}

func removeObligationSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	for _, oId := range modalValues.Obligations {
		intOId, err := strconv.Atoi(oId)
		if err != nil {
			log.Printf("Failed to convert obligation id to int\n")
			return
		}

		handlers.DeleteObligation(intOId)
	}
}

func setGroupSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)
	userIdInt, _ := strconv.Atoi(modalValues.UserID)
	groupIdInt, _ := strconv.Atoi(modalValues.SelectedGroup)

	handlers.SetUserGroup(types.UserGroup{UserID: &userIdInt, GroupID: &groupIdInt})
}

func updateUserSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)
	userIdInt, _ := strconv.Atoi(modalValues.UserID)

	handlers.UpdateUserData(userIdInt, modalValues.DataType, modalValues.NewValue)

	accId, err := handlers.AccountIdByUserID(userIdInt)
	if err != nil {
		log.Printf("error getting accId by userID: %v", err)
	}

	fcm, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID:   &accId,
		Type:        handlers.Pointer("FCM"),
		ExpiredOnly: false,
	})

	for _, token := range fcm {
		handlers.SendFCMNotification(types.SendNotification{
			AccId:  accId,
			UserId: userIdInt,
			FCM:    token.Token,
			Translations: types.Translations{
				Si: types.LangContent{
					Title: "Spremenjeni podatki",
					Body:  "Vaši profilni podatki so bili spremenjeni",
				},
				En: types.LangContent{
					Title: "Profile Changed",
					Body:  "Your profile information has been changed",
				},
				Bs: types.LangContent{
					Title: "Izmijenjeni podaci profila",
					Body:  "Vaši profilni podaci su izmijenjeni",
				},
			},
			Type:      "acknowledgement",
			Update:    "true",
			NavPage:   "/profile-page",
			Arguments: "",
		})
	}
}

func updateObligationStatus(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	userIdInt, err := strconv.Atoi(modalValues.UserID)
	if err != nil {
		log.Printf("Failed to convert userId to int: %v", err)
		return
	}

	if len(modalValues.Obligations) == 0 {
		return
	}

	var obString string

	for _, v := range modalValues.Obligations {
		obligationIdInt, _ := strconv.Atoi(v)

		if len(obString) != 0 {
			obString = fmt.Sprintf("%s, %s", obString, v)
		} else {
			obString = fmt.Sprintf("%s", v)
		}

		_, err := handlers.SetUserObligationStatus(&types.UserObligation{
			UserId:       userIdInt,
			ObligationID: obligationIdInt,
			Status:       modalValues.SelectedStatus,
		})
		if err != nil {
			log.Printf("Failed to set obligation status: %v", err)
			return
		}
	}

	toUpdate := false

	if handlers.CheckCompletionStatus(userIdInt) {
		toUpdate = true
		handlers.AllObligationCompleted(userIdInt)
	}

	accId, err := handlers.AccountIdByUserID(userIdInt)
	if err != nil {
		log.Printf("error getting accId by userID: %v", err)
	}

	fcm, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID:   &accId,
		Type:        handlers.Pointer("FCM"),
		ExpiredOnly: false,
	})

	for _, token := range fcm {
		handlers.SendFCMNotification(types.SendNotification{
			AccId:  accId,
			UserId: userIdInt,
			FCM:    token.Token,
			Translations: types.Translations{
				Si: types.LangContent{
					Title: "Statusi obveznosti spremenjeni",
					Body:  "Statusi obveznosti so bili spremenjeni",
				},
				En: types.LangContent{
					Title: "Updated obligations",
					Body:  "Obligation statuses has been updated",
				},
				Bs: types.LangContent{
					Title: "Ažurirani statusi obaveza",
					Body:  "Statusi obaveza su ažurirani",
				},
			},
			Type:      "acknowledgement",
			Update:    fmt.Sprintf("%t", toUpdate),
			NavPage:   "/list-obligations",
			Arguments: obString,
		})
	}
}

func updateEmploymentStatus(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	userIdInt, err := strconv.Atoi(modalValues.UserID)
	if err != nil {
		log.Printf("Failed to convert userId to int: %v", err)
		return
	}

	err = handlers.UpdateUserJobStatus(userIdInt, modalValues.EmploymentStatus)
	if err != nil {
		log.Printf("update employment status --> %v", err)
	}

	accId, err := handlers.AccountIdByUserID(userIdInt)
	if err != nil {
		log.Printf("error getting accId by userID: %v", err)
	}

	fcm, err := handlers.FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID:   &accId,
		Type:        handlers.Pointer("FCM"),
		ExpiredOnly: false,
	})

	for _, token := range fcm {
		handlers.SendFCMNotification(types.SendNotification{
			AccId:  accId,
			UserId: userIdInt,
			FCM:    token.Token,
			Translations: types.Translations{
				Si: types.LangContent{
					Title: "Vaš status zaposlitve je bil spremenjen",
					Body:  "Status zaposlitve spremenjen",
				},
				En: types.LangContent{
					Title: "Your employment status has been updated",
					Body:  "Employment status updated",
				},
				Bs: types.LangContent{
					Title: "Vaš radni status je ažuriran",
					Body:  "Radni status ažuriran",
				},
			},
			Type:      "acknowledgement",
			Update:    "false",
			NavPage:   "/employment-page",
			Arguments: "",
		})
	}

}

func uploadDocumentSubmission(interaction types.SlackInteractionPayload) {
	modalValues := ExtractModalValues(interaction.View)

	userIdInt, err := strconv.Atoi(modalValues.UserID)
	if err != nil {
		log.Printf("Failed to convert userId to int: %v", err)
		return
	}

	_, err = sendMessage(types.SlackMessage{
		Channel: modalValues.StructuredMetadata.SlackUserId,
		Text:    fmt.Sprintf("✅ Naloži datoteko tipa %s tukaj.", modalValues.DocumentType),
		Blocks:  entityCreatedMessage("document"),
	})
	if err != nil {
		return
	}

	documentSessions[interaction.User.ID] = documentUploadSession{
		Type:        modalValues.DocumentType,
		Title:       modalValues.Translations.Si.Title,
		UserID:      userIdInt,
		SlackUserID: modalValues.StructuredMetadata.SlackUserId,
	}
}
