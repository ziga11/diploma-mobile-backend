package handlers

import (
	"backend/types"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDriveService struct {
	config *oauth2.Config
	srv    *drive.Service
}

func NewGoogleDriveService(refreshToken string) (*GoogleDriveService, error) {
	ctx := context.Background()

	config := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URI"),
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveScope},
	}

	if refreshToken == "" {
		return &GoogleDriveService{config: config, srv: nil}, nil
	}

	token := &oauth2.Token{RefreshToken: refreshToken}

	ts := config.TokenSource(ctx, token)

	httpClient := oauth2.NewClient(ctx, ts)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	return &GoogleDriveService{
		config: config,
		srv:    srv,
	}, nil
}

func (s *GoogleDriveService) ExchangeCode(ctx context.Context, code string) (string, error) {
	tok, err := s.config.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %v", err)
	}

	if tok.RefreshToken == "" {
		return "", fmt.Errorf("no refresh token returned; try re-authorizing with prompt=consent")
	}

	return tok.RefreshToken, nil
}

func (s *GoogleDriveService) GetAuthURL() string {
	return s.config.AuthCodeURL("state-token",
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce)
}

func StartGoogleAuth(w http.ResponseWriter, r *http.Request) {
	driveSvc, err := NewGoogleDriveService("")
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	url := driveSvc.GetAuthURL()

	log.Printf("%s <-- auth url", url)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func SaveGoogleToken(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code missing", http.StatusBadRequest)
		return
	}

	log.Printf("Called save google token")

	driveSvc, err := NewGoogleDriveService("")
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	refreshToken, err := driveSvc.ExchangeCode(r.Context(), code)
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	err = DeleteEncryptedTokens(types.EncryptedTokenLookup{
		Type:        Pointer("google_refresh_token"),
		ExpiredOnly: false,
	})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	err = insertEncryptedToken(types.EncryptedToken{
		AccountID: 1,
		Token:     refreshToken,
		Type:      "google_refresh_token",
		ExpiresAt: time.Now().Add(5 * 365 * 24 * time.Hour),
	})
	if RespondIfErr(w, err, http.StatusInternalServerError) {
		return
	}

	w.Write([]byte("Authenticated! Refresh token saved."))
}

func (s *GoogleDriveService) GetOrCreateDir(folderName string, parentID string) (string, error) {
	query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
		folderName, parentID)

	list, err := s.srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}

	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	fileMetadata := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}

	folder, err := s.srv.Files.Create(fileMetadata).Fields("id").Do()
	if err != nil {
		return "", err
	}

	return folder.Id, nil
}

func (s *GoogleDriveService) UploadFile(content io.Reader, fileName string, contentType string, folderID string) (*drive.File, error) {
	fileMetadata := &drive.File{
		Name:    fileName,
		Parents: []string{folderID},
	}

	upload, err := s.srv.Files.Create(fileMetadata).Media(content).Fields("id, name, mimeType").Do()
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (s *GoogleDriveService) TrashDir(id string) (int, error) {
	update := &drive.File{
		Trashed: true,
	}

	res, err := s.srv.Files.Update(id, update).Do()
	if err != nil {
		return 500, err
	}

	return res.HTTPStatusCode, nil
}
