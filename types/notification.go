package types

import (
	"time"
)

type Notification struct {
	LinkID string `json:"link_id"`
	Type   string `json:"type"`

	Translations Translations `json:"translations"`

	UserID   int       `json:"user_id"`
	Date     time.Time `json:"date"`
	Read     bool      `json:"read"`
	Suitable *bool     `json:"suitable"`
	ThreadTs *string   `json:"thread_ts"`
}

type UserNotification struct {
	NotificationID int       `json:"notification_id"`
	LinkID         int       `json:"link_id"`
	UserID         int       `json:"user_id"`
	Date           time.Time `json:"date"`
	Read           bool      `json:"read"`
	Suitable       *bool     `json:"suitable"`
	ThreadTs       *string   `json:"thread_ts"`
}

type NotificationTranslations struct {
	NotificationID int
	Translations   Translations `json:"translations"`
}

type FCMMessage struct {
	Token        string            `json:"token"`
	Notification *LangContent      `json:"notification"`
	Data         map[string]string `json:"data"`
}

type FCMMessageWrapper struct {
	Message FCMMessage `json:"message"`
}

type SendNotification struct {
	AccId        int
	UserId       int
	FCM          string
	Translations Translations
	Type         string
	ThreadTS     *string
	Update       string
	NavPage      string
	Arguments    string
}
