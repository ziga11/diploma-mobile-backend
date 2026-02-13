package handlers

import (
	"backend/types"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"mime"
	"path"
	"slices"
	"strconv"
	"unicode"

	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/resend/resend-go/v2"
)

var MobileDB *sql.DB
var GoogleDrive *GoogleDriveService
var fcmKey = []byte(os.Getenv("FCM_ENCRYPT_KEY"))

func headerAccId(r *http.Request) (int, error) {
	accountIDStr := r.Header.Get("X-Account-ID")
	aId, err := strconv.Atoi(accountIDStr)
	if err != nil {
		return -1, fmt.Errorf("Failed to get accId from header")
	}

	return aId, nil
}

func StartGoogleDrive() (*GoogleDriveService, error) {
	tokens, err := FetchEncryptedTokens(types.EncryptedTokenLookup{
		AccountID:   Pointer(1),
		Type:        Pointer("google_refresh_token"),
		ExpiredOnly: false,
	})
	if err != nil {
		log.Printf("Failed to fetch google refresh token "+err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	driveSvc, err := NewGoogleDriveService(tokens[0].Token)
	if err != nil {
		log.Printf("Drive service error: "+err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	return driveSvc, nil
}

func AuthToken(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	aId, err := headerAccId(r)
	if err != nil {
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	t, err := MatchingToken(types.HashedTokenLookup{Type: Pointer("secure_token"), Token: &token})

	return err == nil && t.AccountID == aId
}

func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func TernaryOperator[T any](condition bool, option1, option2 T) T {
	if condition {
		return option1
	}
	return option2
}

func ArrayContains(s []string, e string) bool {
	return slices.Contains(s, e)
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func WriteOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func SendEmail(to, subject, emailBody string) error {
	client := resend.NewClient(os.Getenv("RESEND_API_KEY"))

	params := &resend.SendEmailRequest{
		From:    "noreply@zigatdiploma.org",
		To:      []string{to},
		Html:    emailBody,
		Subject: subject,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return fmt.Errorf("Napaka pri pošiljanju maila: %v", err)
	}

	return nil
}

func FirstNChars(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

func RemoveElem[T any](slice []T, s int) []T {
	return append(slice[:s], slice[s+1:]...)
}

func VerifyEmail(email string, envKey string) error {
	baseURL := "https://api.zerobounce.net/v2/validate"
	reqURL, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return fmt.Errorf("Error parsing URL: %v", err)
	}

	resp, err := http.Get(reqURL.String() + fmt.Sprintf("?api_key=%s&email=%s", os.Getenv(envKey), email))
	if err != nil {
		log.Printf("Invalid ZeroBounce email request: %v", err)
		return fmt.Errorf("Invalid ZeroBounce email Request: %v", err)
	}
	defer resp.Body.Close()

	var r types.ZeroBounce
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Printf("Error handling response: %v", err)
		return fmt.Errorf("Error handling response: %v", err)
	}

	if r.Status != "valid" {
		return fmt.Errorf("Email not verified: %v", err)
	}

	return nil
}

func replacePlaceholder(jsonTranslations *types.Translations, obligTranslations types.Translations) {
	jsonTranslations.Si.Title = strings.ReplaceAll(jsonTranslations.Si.Title, "{{obligationName}}", obligTranslations.Si.Title)
	jsonTranslations.En.Title = strings.ReplaceAll(jsonTranslations.En.Title, "{{obligationName}}", obligTranslations.En.Title)
	jsonTranslations.Bs.Title = strings.ReplaceAll(jsonTranslations.Bs.Title, "{{obligationName}}", obligTranslations.Bs.Title)
}

func setReminderBody(obligation *types.Obligation, timeElapsed time.Duration) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("failed to get current file path")
		return
	}
	currentDir := filepath.Dir(filename)
	reminderJson := filepath.Join(currentDir, "../reminders.json")

	b, err := os.ReadFile(reminderJson)
	if err != nil {
		log.Printf("failed to read reminders.json: %v", err)
	}

	var reminders []types.Reminder
	if err := json.Unmarshal(b, &reminders); err != nil {
		log.Printf("failed to unmarshal reminders: %v", err)
	}

	for _, r := range reminders {
		if timeElapsed < time.Duration(r.MaxDays)*24*time.Hour {
			replacePlaceholder(&r.Translations, obligation.Translations)
			obligation.Translations = r.Translations
		}
	}
}

func sendObligationReminder(obligation types.Obligation) {
	accId, err := AccountIdByUserID(obligation.UserID)
	if err != nil {
		return
	}

	tokens, err := FetchEncryptedTokens(types.EncryptedTokenLookup{AccountID: &accId, Type: Pointer("FCM")})
	if err != nil {
		return
	}

	timeElapsed := time.Since(obligation.Date)
	setReminderBody(&obligation, timeElapsed)

	for _, token := range tokens {
		acc, err := AccountById(token.AccountID)
		if err != nil {
			log.Printf("Failed to send group notification: %v", err)
			return
		}

		SendFCMNotification(
			types.SendNotification{
				AccId:        acc.ID,
				UserId:       acc.UserID,
				FCM:          token.Token,
				Translations: obligation.Translations,
				Type:         "acknowledgement",
				Update:       "false",
				NavPage:      "/list-obligations",
				Arguments:    fmt.Sprintf("%d", obligation.Id),
			})
	}

}

func GetStringOrDefault(value any, defaultValue string, allowedNull bool) *string {
	if str, ok := value.(string); ok {
		return &str
	}
	if !allowedNull {
		return nil
	}
	return &defaultValue
}

func ParseNullDate(dateStr string) sql.NullTime {
	if dateStr == "" {
		return sql.NullTime{Valid: false}
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func ParseTimeDate(dateStr string) *time.Time {
	var date time.Time
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Printf("Failed to parse date")
		return nil
	}

	return &date
}

func RespondIfErr(w http.ResponseWriter, err error, code int) bool {
	if err != nil {
		log.Printf("%v: %v", err, code)
		http.Error(w, err.Error(), code)
		return true
	}
	return false
}

func LangContentByCode(translations types.Translations, langCode string) types.LangContent {
	switch langCode {
	case "si":
		return translations.Si
	case "en":
		return translations.En
	case "bs":
		return translations.Bs
	}
	return translations.En
}

func Pointer[T any](s T) *T { return &s }

func getQueryParam[T any](w http.ResponseWriter, r *http.Request, key string) (T, bool) {
	val, err := QueryGetType[T](r, key)
	if RespondIfErr(w, err, http.StatusBadRequest) {
		var zero T
		return zero, false
	}
	return val, true
}

func QueryGetType[T any](r *http.Request, key string) (T, error) {
	var zero T

	val := r.URL.Query().Get(key)
	if val == "" {
		return zero, fmt.Errorf("query parameter %s not found", key)
	}

	switch any(zero).(type) {
	case int:
		v, err := strconv.Atoi(val)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case bool:
		v, err := strconv.ParseBool(val)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case time.Time:
		v, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	case string:
		return any(val).(T), nil
	default:
		return zero, fmt.Errorf("unsupported type")
	}
}

func GetMime(filename string) string {
	t := mime.TypeByExtension(path.Ext(filename))
	if t == "" {
		return "application/octet-stream"
	}
	return t
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func encryptAESGCM(plaintext string) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(fcmKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

func decryptAESGCM(ciphertext, nonce []byte) (string, error) {
	block, err := aes.NewCipher(fcmKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(nonce) != gcm.NonceSize() {
		return "", fmt.Errorf("invalid nonce size: got %d, want %d", len(nonce), gcm.NonceSize())
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	if string(plaintext) == "" {
		return "", fmt.Errorf("deciphered text was empty")
	}

	return string(plaintext), nil
}
