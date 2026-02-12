package handlers

import (
	"backend/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var googleApi = string(os.Getenv("GOOGLE_TRANSLATE_API"))

func Translate(sourceLanguage, targetLanguage string, texts []string) ([]types.Translation, error) {
	requestBody := types.TranslateReq{
		Text:   texts,
		Source: sourceLanguage,
		Target: targetLanguage,
		Format: "text",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2?key=%s", googleApi)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Translate API error: %s", body)
	}

	var result types.TranslationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data.Translations) == 0 {
		return nil, fmt.Errorf("no translation returned")
	}

	return result.Data.Translations, nil
}
