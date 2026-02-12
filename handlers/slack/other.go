package slack

import (
	"backend/handlers"
	"backend/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var slackApi = string(os.Getenv("SLACK_API"))

var slackChannelMap = map[string]string{
	"obligation":   "C089KMEGWJF",
	"request":      "C0917GCDXPW",
	"notification": "C0A63D1S1K7",
}

type obligationTypes struct {
	Obligation    types.Oblg
	Translations  types.ObligationTranslation
	Applicability types.ObligationApplicability
}

type documentUploadSession struct {
	FileID      string
	Title       string
	UserID      int
	Type        string
	SlackUserID string
}

type obligationUploadSession struct {
	Channel     string
	ThreadTs    string
	SlackUserID string
	OTypes      obligationTypes
	FileID      string
}

type requestUploadSession struct {
	Channel      string
	ThreadTs     string
	FCM          string
	SlackUserID  string
	UserID       int
	Translations types.Translations
	FileIDs      []string
	ParentMsgID  *int
}

var obligationSessions = map[string]obligationUploadSession{}
var requestSessions = map[string]requestUploadSession{}
var documentSessions = map[string]documentUploadSession{}

func sendMessage(slackMessage types.SlackMessage) (string, error) {
	url := "https://slack.com/api/chat.postMessage"

	jsonMsg, err := json.Marshal(slackMessage)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return "", fmt.Errorf("Failed to marshal slack message: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonMsg))
	if err != nil {
		log.Printf("failed to create slack request: %v", err)
		return "", fmt.Errorf("Failed to create slack request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+slackApi)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v", err)
		return "", fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var responseBody types.SlackRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Printf("failed to decode Slack response: %v", err)
		return "", fmt.Errorf("Failed to decode slack response: %v", err)
	}

	if !responseBody.OK {
		log.Printf("Slack API error: %s", responseBody.Error)
		return "", fmt.Errorf("Slack API error: %v", err)
	}

	return responseBody.TS, nil
}

func craftMessage(title, body string) string {
	var text string

	if len(title) > 0 {
		text = fmt.Sprintf("*%s*", title)
	}
	if len(body) > 0 {
		text = fmt.Sprintf("%s\n\n```\n%s\n```", text, body)
	}

	return text
}

func SendMessageReq(w http.ResponseWriter, r *http.Request) {
	var sMsg types.SlackClientMessage
	if err := json.NewDecoder(r.Body).Decode(&sMsg); err != nil {
		http.Error(w, "Failed to decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	text := craftMessage(sMsg.Title, sMsg.Body)

	modalValues := types.StructuredModalValues{
		UserId:       sMsg.UserID,
		Channel:      sMsg.Channel,
		MessageId:    sMsg.MessageID,
		ObligationId: sMsg.ObligationID,
		FcmToken:     sMsg.FCMToken,
		Text:         &text,
	}

	threadTS, err := sendMessage(types.SlackMessage{
		Channel: slackChannelMap[sMsg.Channel],
		Text:    text,
		Blocks:  slackMessage(modalValues, sMsg.Channel != "notification"),
	})
	if handlers.RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(map[string]any{"thread_ts": threadTS})
	handlers.RespondIfErr(w, err, http.StatusInternalServerError)
}

func parseKeyValuePairs(r *http.Request) (types.FileRequest, error) {
	title := r.FormValue("title")
	body := r.FormValue("body")
	fcm := r.FormValue("fcm")
	channel := r.FormValue("channel")

	threadTs := r.FormValue("threadTs")

	userId := r.FormValue("user_id")
	uIdInt, err := strconv.Atoi(userId)
	if err != nil {
		return types.FileRequest{}, err
	}

	mId := r.FormValue("message_id")
	mIdInt, err := strconv.Atoi(mId)
	if err != nil {
		mIdInt = -1
	}

	oId := r.FormValue("obligation_id")
	oIdInt, err := strconv.Atoi(oId)
	if err != nil {
		oIdInt = -1
	}

	return types.FileRequest{
		UserID:       uIdInt,
		FCMToken:     fcm,
		Title:        title,
		Body:         body,
		Channel:      channel,
		MessageID:    &mIdInt,
		ObligationID: &oIdInt,
		ThreadTS:     handlers.TernaryOperator(threadTs == "", nil, &threadTs),
	}, nil
}

func uploadFile(file multipart.File, filename string, size int64) (string, error) {
	uploadURL, slackFileID, err := getUploadURL(filename, size)
	if err != nil {
		return "", err
	}

	uploadReq, _ := http.NewRequest("POST", uploadURL, file)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadReq.ContentLength = size

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil || uploadResp.StatusCode != 200 {
		return "", fmt.Errorf("Failed to stream to slack :%v", err)
	}
	uploadResp.Body.Close()

	return slackFileID, nil
}

func AttachFilesToSlack(w http.ResponseWriter, r *http.Request) {
	fq, err := parseKeyValuePairs(r)
	if handlers.RespondIfErr(w, err, http.StatusBadRequest) {
		return
	}

	if !handlers.AuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(25 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["files"]
	var slackFiles []types.ReceiveSlackFile

	for _, header := range files {
		f, _ := header.Open()

		slackFileID, err := uploadFile(f, header.Filename, header.Size)
		f.Close()

		if err == nil {
			slackFiles = append(slackFiles, types.ReceiveSlackFile{
				ID:    slackFileID,
				Title: header.Filename,
			})
		}
	}

	threadTs, err := completeFileUpload(slackFiles, "", slackChannelMap[fq.Channel])
	if handlers.RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	text := craftMessage(fq.Title, fq.Body)

	_, err = sendMessage(types.SlackMessage{
		Channel:  slackChannelMap[fq.Channel],
		ThreadTS: handlers.TernaryOperator(fq.ThreadTS == nil, &threadTs, fq.ThreadTS),
		Blocks: slackMessage(
			types.StructuredModalValues{
				UserId:       fq.UserID,
				Channel:      fq.Channel,
				MessageId:    fq.MessageID,
				ObligationId: fq.ObligationID,
				FcmToken:     fq.FCMToken,
				Text:         &text,
			}, true),
	})

	if handlers.RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(threadTs)
	handlers.RespondIfErr(w, err, http.StatusInternalServerError)
}

func completeFileUpload(files []types.ReceiveSlackFile, comment, channel string) (string, error) {
	url := "https://slack.com/api/files.completeUploadExternal"

	payload := types.SlackFilePayload{
		Files:          files,
		InitialComment: comment,
		ChannelId:      channel,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling request: %v", err)
		return "", fmt.Errorf("Error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("error creating request: %v", err)
		return "", fmt.Errorf("Error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+slackApi)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v", err)
		return "", fmt.Errorf("Error sending request: %v", err)
	}
	defer res.Body.Close()

	var response types.SlackFileUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("Error decoding response: %v", err)
	}

	if !response.OK {
		log.Printf("Slack API error multifile upload: %s", response.Error)
		return "", fmt.Errorf("Slack API error with multifile upload: %v", err)
	}

	return waitForFileShareTS(files[0].ID, channel), nil
}

func waitForFileShareTS(fileID, channel string) string {
	for range 15 {
		time.Sleep(1 * time.Second)

		req, _ := http.NewRequest("GET", "https://slack.com/api/files.info?file="+fileID, nil)
		req.Header.Set("Authorization", "Bearer "+slackApi)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("error requesting file info: %v", err)
			continue
		}
		defer res.Body.Close()

		var infoResp types.SlackFileUploadResponse
		if err := json.NewDecoder(res.Body).Decode(&infoResp); err != nil {
			log.Printf("error decoding file info: %v", err)
			continue
		}

		if shares, ok := infoResp.File.Shares.Public[channel]; ok && len(shares) > 0 {
			return shares[0].TS
		}
	}

	log.Printf("Timed out waiting for file to appear in channel %s", channel)
	return ""
}

func fileInfo(fileId string) (types.FileInfoRequestResponse, error) {
	data := url.Values{"file": []string{fileId}}
	uri := "https://slack.com/api/files.info"

	req, _ := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	req.Header.Add("Authorization", "Bearer "+slackApi)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("files.info error:", err)
		return types.FileInfoRequestResponse{}, err
	}
	defer resp.Body.Close()

	var fileResp types.FileInfoRequestResponse

	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return types.FileInfoRequestResponse{}, err
	}

	return fileResp, nil
}

func slackFileStream(url string) (io.ReadCloser, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Add("Authorization", "Bearer "+slackApi)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func uploadSlackFileToDrive(fileID string, slackUserId string) (string, string, error) {
	fileResp, err := fileInfo(fileID)
	if err != nil || !fileResp.OK {
		errMsg := handlers.TernaryOperator(err != nil, "internal error", fileResp.Error)
		return "", "", fmt.Errorf("slack info error: %v", errMsg)
	}

	mimetype := fileResp.File.MimeType

	body, _, err := slackFileStream(fileResp.File.URLPrivate)
	if err != nil {
		return "", "", err
	}
	defer body.Close()

	rootFolderID := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
	folderId, err := handlers.GoogleDrive.GetOrCreateDir(slackUserId, rootFolderID)
	if err != nil {
		return "", "", err
	}

	driveFile, err := handlers.GoogleDrive.UploadFile(
		body,
		fileResp.File.Name,
		mimetype,
		folderId,
	)

	return handlers.TernaryOperator(err != nil, "", driveFile.Id), mimetype, err
}

func updateModal(payload types.UpdatePayload) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/views.update", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+slackApi)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return
	}
	defer resp.Body.Close()

	var result types.SlackRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to decode modal response: %v %s", err, result.Error)
		return
	}

	if !result.OK {
		log.Printf("Slack API error opening modal: %v %s", err, result.Error)
		return
	}
}

func updateMessage(responseURL string, message types.MessageUpdatePayload) {
	payloadBytes, _ := json.Marshal(message)

	resp, err := http.Post(responseURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error updating message: %v", err)
		return
	}
	defer resp.Body.Close()
}

func openModal(payload *types.SlackModalPayload) {
	url := "https://slack.com/api/views.open"

	jsonPayload, err := json.Marshal(*payload)
	if err != nil {
		log.Printf("failed to marshal modal payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("failed to create modal request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+slackApi)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send modal request: %v", err)
		return
	}
	defer resp.Body.Close()

	var result types.SlackRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to decode modal response: %v", err)
		return
	}

	if !result.OK {
		log.Printf("Slack API error opening modal: %s", result.Error)
		return
	}
}

func hideClickedButton(interaction types.SlackInteractionPayload) {
	update := types.MessageUpdatePayload{
		ReplaceOriginal: true,
		Text:            "Procesiranje zahteve",
	}

	updateMessage(interaction.ResponseURL, update)
}
