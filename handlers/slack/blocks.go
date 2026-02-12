package slack

import (
	"backend/types"
	"encoding/json"
	"fmt"
)

func entityCreatedMessage(entityType string) []types.Block {
	var entityTypeSi string
	switch entityType {
	case "obligation":
		entityTypeSi = "obveznosti"
	case "document":
		entityTypeSi = "dokumenta"
	default:
		entityTypeSi = "prošnje"
	}

	return []types.Block{
		types.SectionBlock{
			Type: "section",
			Text: &types.SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf(
					"✅ Dokončajte ustvarjanje %s s klikom na spodnji gumb *Dokončano*.\n\n📎 Če želite dodati kakršen koli dokument, ga naložite v klepet in nato kliknite *Dokončano*.", entityTypeSi),
			},
		},
		types.ActionBlock{
			Type: "actions",
			Elements: []types.ModalActionElement{
				{
					Type: "button",
					Text: types.SlackText{
						Type: "plain_text",
						Text: "Dokončano ✅",
					},
					ActionID: fmt.Sprintf("finished_%s_upload", entityType),
					Value:    "done",
				},
			},
		},
	}
}

func savedFileMessage(message, entityType string) []types.Block {
	return []types.Block{
		types.SectionBlock{
			Type: "section",

			Text: &types.SlackText{
				Type: "mrkdwn",
				Text: message,
			},
		},
		types.ActionBlock{
			Type: "actions",
			Elements: []types.ModalActionElement{
				{
					Type: "button",
					Text: types.SlackText{
						Type: "plain_text",
						Text: "Dokončano ✅",
					},
					ActionID: fmt.Sprintf("finished_%s_upload", entityType),
					Value:    "done",
				},
			},
		},
	}
}

func fileUploadFinished(entityType string) []types.Block {
	return []types.Block{types.SectionBlock{
		Type: "section",
		Text: &types.SlackText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("🎉 *Končano!* Vaš/a %s je bil/a uspešno predelan/a.", entityType),
		},
	}}
}

func slackMessage(modalValues types.StructuredModalValues, buttons bool) []types.Block {
	valueBytes, _ := json.Marshal(modalValues)
	buttonValue := string(valueBytes)

	blocks := []types.Block{
		types.SectionBlock{
			Type: "section",
			Text: &types.SlackText{Type: "mrkdwn", Text: *modalValues.Text},
		},
		types.SectionBlock{
			Type: "divider",
		},
	}

	if buttons {
		actions := types.ActionBlock{
			Type: "actions",
			Elements: []types.ModalActionElement{
				{
					Type:     "button",
					Text:     types.SlackText{Type: "plain_text", Text: "Potrdi"},
					ActionID: "accept_action",
					Value:    buttonValue,
					Style:    "primary",
				},
				{
					Type:     "button",
					Text:     types.SlackText{Type: "plain_text", Text: "Zavrni"},
					ActionID: "reject_action",
					Value:    buttonValue,
					Style:    "danger",
				},
			},
		}
		blocks = append(blocks, actions)
	}

	return blocks
}

func listUserObligationsBlock(userObligations []types.Obligation) types.Block {
	slackOptions := make([]types.SlackOption, len(userObligations))
	for i, obligation := range userObligations {
		slackOptions[i] = createOption(obligation.Translations.Si.Title, fmt.Sprintf("%d", obligation.Id))
	}

	return types.InputBlock{
		Type:    "input",
		BlockID: "obligation_select_block",
		Label: types.SlackText{
			Type: "plain_text",
			Text: "Izberite dolžnosti:",
		},
		Element: types.ModalInputElement{
			Type:     "multi_static_select",
			ActionID: "obligation_select",
			Placeholder: types.SlackText{
				Type: "plain_text",
				Text: "Izberite možnosti...",
			},
			Options: slackOptions,
		},
	}
}

func obligationStatuses() types.Block {
	nameArray := []string{"Nepopolno", "Čakanje", "Dokončano"}
	valueArray := []string{"Incomplete", "Pending", "Completed"}

	slackOptions := make([]types.SlackOption, len(nameArray))
	for i := range len(valueArray) {
		slackOptions[i] = createOption(nameArray[i], valueArray[i])
	}

	return types.InputBlock{
		Type:    "input",
		BlockID: "status_block",
		Label: types.SlackText{
			Type: "plain_text",
			Text: "Izberite status:",
		},
		Element: types.ModalInputElement{
			Type:     "static_select",
			ActionID: "status_select",
			Placeholder: types.SlackText{
				Type: "plain_text",
				Text: "Izberite možnost...",
			},
			Options: slackOptions,
		},
	}
}
