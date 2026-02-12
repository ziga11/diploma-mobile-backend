package types

import (
	"time"
)

type Message struct {
	ID          int       `json:"id"`
	Date        time.Time `json:"date"`
	Sender      User      `json:"sender"`
	ParentMsgID *int      `json:"parent_msg_id"`
	ThreadTS    *string   `json:"thread_ts"`

	Read bool `json:"read"`

	Translations Translations `json:"translations"`
	Recipient    User         `json:"recipient"`
	DocIds       []int        `json:"document_ids"`
}

type Msg struct {
	ID       int       `json:"id"`
	Date     time.Time `json:"date"`
	Sender   User      `json:"sender"`
	ParentId *int      `json:"parent_msg_id"`
	ThreadTS *string   `json:"thread_ts"`
}

type UserMessage struct {
	ID        int
	UserID    int  `json:"user_id"`
	MessageID *int `json:"message_id"`
	Read      bool `json:"read"`
}

type MessageTranslations struct {
	MessageID    int          `json:"message_id"`
	Translations Translations `json:"translations"`
}

type MessageDocuments struct {
	MessageID int `json:"message_id"`
	DocId     int `json:"document_id"`
}
