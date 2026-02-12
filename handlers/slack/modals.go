package slack

import (
	"backend/handlers"
	"backend/types"
	"encoding/json"
	"fmt"
)

func createOption(text, value string) types.SlackOption {
	return types.SlackOption{
		Text: types.SlackText{
			Type: "plain_text",
			Text: text,
		},
		Value: value,
	}
}

func indexOfName(groups []types.Group, name string) int {
	for i, v := range groups {
		if v.Name == name {
			return i
		}
	}

	return -1
}

func notificationModal(groups []types.Group, triggerID string) *types.SlackModalPayload {
	indexOfAdmin := indexOfName(groups, "Admin")

	groups = handlers.RemoveElem(groups, indexOfAdmin)

	slackGroupArr := make([]types.SlackOption, len(groups))
	for i, group := range groups {
		slackGroupArr[i] = createOption(group.Name, fmt.Sprintf("%d", group.ID))
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "notification_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Množično obvestilo",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "group_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi skupino:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "category_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: slackGroupArr,
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Vnesi naslov (si)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_si",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi naslov tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (si)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_si",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi sporočilo tukaj...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "translate_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "translate_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Prevedi",
							},
							Value: "translate",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Title (en)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_en",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Enter your title here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Message (en)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_en",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Type your message here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (bs)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_bs",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi naslov tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (bs)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_bs",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi sporočilo tukaj...",
						},
					},
				},
				types.SectionBlock{
					Type:    "section",
					BlockID: "message_type_block",
					Text: &types.SlackText{
						Type: "mrkdwn",
						Text: "Izberi tip sporočila:",
					},
					Accessory: &types.RadioButtonElement{
						Type:     "radio_buttons",
						ActionID: "message_type_selection",
						Options: []types.SlackOption{
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Seznanjen sporočilo",
								},
								Value: "acknowledgement",
							},
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Ustreza/Neustreza sporočilo",
								},
								Value: "response",
							},
						},
					},
				},
			},
		},
	}
}

func createObligationModal(ujcs []types.UserJobCompany, countries []types.Country, slackUserID, triggerID string) *types.SlackModalPayload {
	workpermitArray := []string{"Brez", "Potrditev", "Začasno", "Ima", "EU in EFTA"}
	workpermitValArray := []string{"NONE", "CONFIRM", "TEMPORARY", "HAS", "EU_EFTA"}

	workpermitSlackOptions := make([]types.SlackOption, len(workpermitArray))
	countrySlackOptions := make([]types.SlackOption, len(countries))
	jobSlackOptions := make([]types.SlackOption, len(ujcs))

	for i, c := range countries {
		countrySlackOptions[i] = createOption(c.Name, fmt.Sprintf("%d", c.ID))
	}

	for i, ujc := range ujcs {
		optionName := fmt.Sprintf("%s(%s)", ujc.JobTitle, ujc.CompanyName)
		optionValue := fmt.Sprintf("%d", ujc.JobID)

		jobSlackOptions[i] = createOption(optionName, optionValue)
	}

	for i, name := range workpermitArray {
		workpermitSlackOptions[i] = createOption(name, workpermitValArray[i])
	}

	structuredValues := types.StructuredModalValues{
		SlackUserId: slackUserID,
	}

	valueBytes, _ := json.Marshal(structuredValues)
	metadata := string(valueBytes)

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Metadata:   metadata,
			Type:       "modal",
			CallbackID: "create_obligation_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Ustvari dolžnost",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "citizenship_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi državljanstva:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "citizenship_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: countrySlackOptions,
					},
					Optional: true,
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "workpermit_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Status delovnega dovoljenja:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "workpermit_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: workpermitSlackOptions,
					},
					Optional: true,
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "job_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Delovno mesto:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "job_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: jobSlackOptions,
					},
					Optional: true,
				},
				types.SectionBlock{
					Type:    "section",
					BlockID: "uploadable_block",
					Text: &types.SlackText{
						Type: "mrkdwn",
						Text: "Nalaganje datotek:",
					},
					Accessory: &types.RadioButtonElement{
						Type: "checkboxes",
						Options: []types.SlackOption{
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Naložljivo",
								},
								Value: "Uploadable",
							},
						},
						ActionID: "uploadable_checkbox",
					},
				},
				types.SectionBlock{
					Type:    "section",
					BlockID: "textfield_block",
					Text: &types.SlackText{
						Type: "mrkdwn",
						Text: "Vnosa teksta:",
					},
					Accessory: &types.RadioButtonElement{
						Type: "checkboxes",
						Options: []types.SlackOption{
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Tekstovno polje",
								},
								Value: "Textfield",
							},
						},
						ActionID: "textfield_checkbox",
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (si)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "title_input_si",
						MaxLength: 75,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi ime tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Opis (si)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_si",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi opis tukaj...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "translate_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "translate_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Prevedi",
							},
							Value: "translate",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Title (en)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "title_input_en",
						MaxLength: 75,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Enter your obligation name here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Description (en)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_en",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Type your description here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Ime (bs)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "title_input_bs",
						MaxLength: 75,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi ime tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Opis (bs)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_bs",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi opis tukaj...",
						},
					},
				},
			},
		},
	}
}

func rejectionModal(structuredValues types.StructuredModalValues) *types.SlackModalPayload {
	valueBytes, _ := json.Marshal(structuredValues)
	metadata := string(valueBytes)

	var siTitle, siPlaceholder string
	if structuredValues.Channel == "obligation" {
		siTitle = "Nepopolna obveznost"
		siPlaceholder = "Vaša oddana obveznost ni ustrezna (npr. nečitljiva slika, napačen podatek)."
	} else {
		siTitle = "Zavrnitev prošnje"
		siPlaceholder = "Vaša prošnja je bila zavrnjena."
	}

	var bsTitle, bsPlaceholder string
	if structuredValues.Channel == "obligation" {
		bsTitle = "Nepotpuna obaveza"
		bsPlaceholder = "Vaša obaveza nije potpuna (npr. dokument nije čitljiv)."
	} else {
		bsTitle = "Odbijanje zahtjeva"
		bsPlaceholder = "Vaš zahtjev je odbijen."
	}

	var enTitle, enPlaceholder string
	if structuredValues.Channel == "obligation" {
		enTitle = "Incomplete obligation"
		enPlaceholder = "Your submission is incomplete or the document is not legible."
	} else {
		enTitle = "Request rejected"
		enPlaceholder = "Your request has been rejected."
	}

	return &types.SlackModalPayload{
		TriggerID: *structuredValues.TriggerId,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "rejection_modal",
			Metadata:   metadata,
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Obrazložitev zavrnitve",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (si)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_si",
						InitialValue: siTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: siPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Razlog zavrnitve:",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_si",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text", Text: "Vnesi razlog tukaj...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "translate_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "translate_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Prevedi",
							},
							Value: "translate",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Title (en)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_en",
						InitialValue: enTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: enPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Rejection reason (en)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_en",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text", Text: "Enter your reason here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (bs)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_bs",
						InitialValue: bsTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: bsPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Razlog odbijanja (bs)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_bs",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text", Text: "Unesite razlog odvdje...",
						},
					},
				},
			},
		},
	}
}

func acceptModal(structuredValues types.StructuredModalValues) *types.SlackModalPayload {
	valueBytes, _ := json.Marshal(structuredValues)
	metadata := string(valueBytes)

	siChannel := handlers.TernaryOperator(structuredValues.Channel == "obligation", "obveznosti", "prošnje")

	var siTitle, siPlaceholder string
	if structuredValues.Channel == "obligation" {
		siTitle = "Potrditev oddaje dolžnosti"
		siPlaceholder = "Vaša oddana obveznost (dokument/podatek) je bila pregledana in potrjena."
	} else {
		siTitle = "Odobritev prošnje"
		siPlaceholder = "Vaša prošnja (obrazec/sprememba) je bila odobrena."
	}

	var bsTitle, bsPlaceholder string
	if structuredValues.Channel == "obligation" {
		bsTitle = "Potvrda izvršene obaveze"
		bsPlaceholder = "Vaša obaveza (dokument/podatak) je pregledana i potvrđena."
	} else {
		bsTitle = "Odobrenje zahtjeva"
		bsPlaceholder = "Vaš zahtjev (obrazac/promjena) je odobren."
	}

	var enTitle, enPlaceholder string
	if structuredValues.Channel == "obligation" {
		enTitle = "Obligation fulfillment confirmed"
		enPlaceholder = "Your submitted obligation (document/data) has been reviewed and confirmed."
	} else {
		enTitle = "Request approved"
		enPlaceholder = "Your request (form/change) has been approved."
	}

	return &types.SlackModalPayload{
		TriggerID: *structuredValues.TriggerId,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "accept_modal",
			Metadata:   metadata,
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Odobritev " + siChannel,
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (si)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_si",
						InitialValue: siTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: siPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (si)",
					},
					Optional: true,
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_si",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi spročilo (lahko prazno)...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "translate_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "translate_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Prevedi",
							},
							Value: "translate",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Title (en)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_en",
						InitialValue: enTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: enPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Message (en)",
					},
					Optional: true,
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_en",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Enter message (can be empty) ...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (bs)",
					},
					Element: types.ModalInputElement{
						Type:         "plain_text_input",
						ActionID:     "title_input_bs",
						InitialValue: bsTitle,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: bsPlaceholder,
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (bs)",
					},
					Optional: true,
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_bs",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Unesite poruku (može biti praznu) ...",
						},
					},
				},
			},
		},
	}
}

func employmentStatusModal(triggerID string) *types.SlackModalPayload {
	statusArr := []string{
		"Kartica čaka na upravni enoti",
		"Mail z dokumentacijo",
		"Napoten na zdravstveni pregled",
		"Oddaja prstnih odtisov",
		"Ponovni zdravstveni pregled",
		"Potrditev s podjetja",
		"Prejeli dokumentacijo",
		"Vloga na ambasadi",
		"Vloga na vašem zavodu za zaposlovanju",
		"Vloga na zavodu za zaposlovanje v Sloveniji",
		"Vloga na zavodu za zdravstveno zavarovanje",
		"Vloga oddana na upravno enoto",
		"Zaposlen",
	}

	valueArr := []string{
		"stCardWaitingUe",
		"stMailDocs",
		"stMedExam",
		"stFingerprints",
		"stMedExamRepeat",
		"stCompanyConfirm",
		"stDocsReceived",
		"stEmbassy",
		"stEmplOfficeLocal",
		"stEmplOfficeSi",
		"stHealthIns",
		"stUeSubmitted",
		"stEmployed",
	}

	slackOptions := make([]types.SlackOption, len(statusArr))
	for i := range len(statusArr) {
		slackOptions[i] = createOption(statusArr[i], valueArr[i])
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "employment_status_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Status zaposlitve",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					BlockID: "employment_status_block",
					Type:    "input",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Status zaposlitve:",
					},
					Element: types.ModalInputElement{
						Type:     "static_select",
						ActionID: "employment_status_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi novi status...",
						},
						Options: slackOptions,
					},
				},
			},
		},
	}
}

func messageUserModal(triggerID string) *types.SlackModalPayload {
	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "message_user_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Sporočilo",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (si)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_si",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi naslov tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (si)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_si",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi sporočilo tukaj...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "translate_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "translate_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Prevedi",
							},
							Value: "translate",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Title (en)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_en",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Enter your title here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_en",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Message (en)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_en",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Type your message here...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov (bs)",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_bs",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi naslov tukaj...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "body_block_bs",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Sporočilo (bs)",
					},
					Element: types.ModalInputElement{
						Type:      "plain_text_input",
						ActionID:  "body_input_bs",
						Multiline: true,
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi sporočilo tukaj...",
						},
					},
				},
				types.SectionBlock{
					Type:    "section",
					BlockID: "message_type_block",
					Text: &types.SlackText{
						Type: "mrkdwn",
						Text: "Tip sporočila:",
					},
					Accessory: &types.RadioButtonElement{
						Type:     "radio_buttons",
						ActionID: "message_type_selection",
						Options: []types.SlackOption{
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Acknowledge message",
								},
								Value: "acknowledgement",
							},
							{
								Text: types.SlackText{
									Type: "plain_text",
									Text: "Accept/Decline message",
								},
								Value: "response",
							},
						},
					},
				},
			},
		},
	}
}

func uploadDocumentModal(triggerID, slackUserID string) *types.SlackModalPayload {
	sloveneTypes := []string{"Plačilna lista", "Pogodba", "Ostalo"}
	englishTypes := []string{"payroll", "contract", "other"}

	slackOptions := make([]types.SlackOption, len(sloveneTypes))
	for i := range len(sloveneTypes) {
		slackOptions[i] = createOption(sloveneTypes[i], englishTypes[i])
	}

	structuredValues := types.StructuredModalValues{
		SlackUserId: slackUserID,
	}

	valueBytes, _ := json.Marshal(structuredValues)
	metadata := string(valueBytes)

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Metadata:   metadata,
			Type:       "modal",
			CallbackID: "upload_document_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Nalganje dokumenta",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "title_block_si",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Naslov dokumenta",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "title_input_si",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi naslov tukaj...",
						},
					},
				},
				types.InputBlock{
					BlockID: "document_type_block",
					Type:    "input",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Tip dokumenta",
					},
					Element: types.ModalInputElement{
						Type:     "static_select",
						ActionID: "document_type_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Tip dokumenta",
						},
						Options: slackOptions,
					},
				},
			},
		},
	}
}

func setObligationStatusModal(triggerID string) *types.SlackModalPayload {
	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "obligation_status_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Določi status dolžnosti",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.ActionBlock{
					Type:    "actions",
					BlockID: "update_modal_block",
					Elements: []types.ModalActionElement{
						{
							Type:     "button",
							ActionID: "update_button",
							Text: types.SlackText{
								Type: "plain_text",
								Text: "Posodobi",
							},
							Value: "posodobi",
						},
					},
				},
			},
		},
	}
}

func assignObligationModal(triggerID string, options []types.ObligationTranslation) *types.SlackModalPayload {
	slackOptions := make([]types.SlackOption, len(options))
	for i, obligation := range options {
		slackOptions[i] = createOption(obligation.Translations.Si.Title, fmt.Sprintf("%d", obligation.ObligationID))
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "assign_obligation_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Dodeli dolžnost",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "obligation_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi dolžnosti:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "obligation_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: slackOptions,
					},
				},
			},
		},
	}
}

func removeObligationModal(triggerID string, options []types.ObligationTranslation) *types.SlackModalPayload {
	slackOptions := make([]types.SlackOption, len(options))
	for i, obligation := range options {
		slackOptions[i] = createOption(obligation.Translations.Si.Title, fmt.Sprintf("%d", obligation.ObligationID))
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "remove_obligation_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Odstrani dolžnost",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "obligation_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Dolžnost za odstranitev:",
					},
					Element: types.ModalInputElement{
						Type:     "multi_static_select",
						ActionID: "obligation_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi možnosti...",
						},
						Options: slackOptions,
					},
				},
			},
		},
	}
}

func updateUserDataModal(triggerID string) *types.SlackModalPayload {
	userOptions := []string{"Ime", "Priimek", "Naslov Prebivališča", "Državljanstvo"}
	userOptionValues := []string{"firstname", "lastname", "address", "country"}

	slackOptions := make([]types.SlackOption, len(userOptions))
	for i, name := range userOptions {
		slackOptions[i] = createOption(name, userOptionValues[i])
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "update_user_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Posodobitev uporabnika",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "data_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Tip spremembe",
					},
					Element: types.ModalInputElement{
						Type:     "static_select",
						ActionID: "data_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Izberi opcijo",
						},
						Options: slackOptions,
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "new_value_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Nova vrednost:",
					},
					Element: types.ModalInputElement{
						Type:     "plain_text_input",
						ActionID: "new_value_input",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Vnesi novo vrednost tukaj ...",
						},
					},
				},
			},
		},
	}
}

func setUserGroupModal(groups []types.Group, triggerID string) *types.SlackModalPayload {
	slackOptions := make([]types.SlackOption, len(groups))
	for i, group := range groups {
		slackOptions[i] = createOption(group.Name, fmt.Sprintf("%d", group.ID))
	}

	return &types.SlackModalPayload{
		TriggerID: triggerID,
		View: types.SlackModalView{
			Type:       "modal",
			CallbackID: "set_group_modal",
			Title: types.SlackText{
				Type: "plain_text",
				Text: "Določitev skupine",
			},
			Close: &types.SlackText{
				Type: "plain_text",
				Text: "Prekliči",
			},
			Submit: &types.SlackText{
				Type: "plain_text",
				Text: "Potrdi",
			},
			Blocks: []types.Block{
				types.InputBlock{
					Type:    "input",
					BlockID: "user_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Izberi kandidata:",
					},
					Element: types.ModalInputElement{
						Type:     "external_select",
						ActionID: "user_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Išči kandidata...",
						},
					},
				},
				types.InputBlock{
					Type:    "input",
					BlockID: "group_select_block",
					Label: types.SlackText{
						Type: "plain_text",
						Text: "Nova skupina",
					},
					Element: types.ModalInputElement{
						Type:     "static_select",
						ActionID: "group_select",
						Placeholder: types.SlackText{
							Type: "plain_text",
							Text: "Nova skupina",
						},
						Options: slackOptions,
					},
				},
			},
		},
	}
}
