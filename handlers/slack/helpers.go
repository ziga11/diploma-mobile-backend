package slack

import (
	"backend/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func modalValue(state types.SlackState, blockID, actionID string) string {
	if blockVals, exists := state.Values[blockID]; exists {
		if actionVal, exists := blockVals[actionID]; exists {
			return actionVal.Value
		}
	}
	return ""
}

func modalSelectedOptions(state types.SlackState, blockID, actionID string) []string {
	var results []string
	if blockVals, exists := state.Values[blockID]; exists {
		if actionVal, exists := blockVals[actionID]; exists && actionVal.SelectedOptions != nil {
			for _, option := range actionVal.SelectedOptions {
				clean := strings.ReplaceAll(option.Value, "_", " ")
				results = append(results, clean)
			}
		}
	}
	return results
}

func modalSelectedOptionValue(state types.SlackState, blockID, actionID string) string {
	if blockVals, exists := state.Values[blockID]; exists {
		if actionVal, exists := blockVals[actionID]; exists && actionVal.SelectedOption != nil {
			return actionVal.SelectedOption.Value
		}
	}
	return ""
}

func ExtractModalValues(view *types.SlackView) types.ModalValues {
	state := view.State

	var structuredValues types.StructuredModalValues
	if view.Metadata != "" {
		if err := json.Unmarshal([]byte(view.Metadata), &structuredValues); err != nil {
			log.Println("Failed to unmarshal the action values!")
		}
	}

	values := types.ModalValues{
		Translations:        types.Translations{},
		SelectedStatus:      modalSelectedOptionValue(state, "status_block", "status_select"),
		SelectedGroup:       modalSelectedOptionValue(state, "group_select_block", "group_select"),
		MessageType:         modalSelectedOptionValue(state, "message_type_block", "message_type_selection"),
		UserID:              modalSelectedOptionValue(state, "user_select_block", "user_select"),
		DataType:            modalSelectedOptionValue(state, "data_select_block", "data_select"),
		DocumentType:        modalSelectedOptionValue(state, "document_type_block", "document_type_select"),
		EmploymentStatus:    modalSelectedOptionValue(state, "employment_status_block", "employment_status_select"),
		SelectedGroups:      modalSelectedOptions(state, "group_block", "category_select"),
		Obligations:         modalSelectedOptions(state, "obligation_select_block", "obligation_select"),
		SelectedCountries:   modalSelectedOptions(state, "citizenship_block", "citizenship_select"),
		SelectedWorkpermits: modalSelectedOptions(state, "workpermit_block", "workpermit_select"),
		SelectedJobs:        modalSelectedOptions(state, "job_block", "job_select"),
		NewValue:            modalValue(state, "new_value_block", "new_value_input"),
		Uploadable:          modalValue(state, "uplodable_block", "uplodable_checkbox") == "Uploadable",
		TextField:           modalValue(state, "textfield_block", "textfield_checkbox") == "Textfield",
		StructuredMetadata:  structuredValues,
	}

	langBlocks := []string{"si", "en", "bs"}
	for _, lang := range langBlocks {
		tBlock, tAction := "title_block_"+lang, "title_input_"+lang
		bBlock, bAction := "body_block_"+lang, "body_input_"+lang

		title := modalValue(state, tBlock, tAction)
		if title == "" {
			title = modalValue(state, tBlock+"_v2", tAction+"_v2")
		}

		body := modalValue(state, bBlock, bAction)
		if body == "" {
			body = modalValue(state, bBlock+"_v2", bAction+"_v2")
		}

		if title != "" || body != "" {
			content := types.LangContent{Title: title, Body: body}
			switch lang {
			case "si":
				values.Translations.Si = content
			case "en":
				values.Translations.En = content
			case "bs":
				values.Translations.Bs = content
			}
		}
	}

	return values
}

func uploadFileToURL(uploadURL string, file *os.File) error {
	file.Seek(0, 0)

	req, err := http.NewRequest("POST", uploadURL, file)
	if err != nil {
		return fmt.Errorf("error creating upload request: %v", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading file: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status: %d", res.StatusCode)
	}

	return nil
}

func getUploadURL(filename string, filesize int64) (string, string, error) {
	uri := "https://slack.com/api/files.getUploadURLExternal"

	data := url.Values{}
	data.Set("filename", filename)
	data.Set("length", fmt.Sprintf("%d", filesize))

	req, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+slackApi)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	var response types.SlackExtUrlResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", "", err
	}

	if !response.OK {
		return "", "", fmt.Errorf("slack api error: %s", response.Error)
	}

	return response.UploadURL, response.FileID, nil
}
