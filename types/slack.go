package types

import (
	"encoding/json"
)

type FileRequest struct {
	UserID       int     `json:"user_id"`
	MessageID    *int    `json:"message_id"`
	ObligationID *int    `json:"obligation_id"`
	FCMToken     string  `json:"fcm_token"`
	Channel      string  `json:"channel"`
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	ThreadTS     *string `json:"thread_ts"`
}

type SlackClientMessage struct {
	MessageID    *int    `json:"message_id"`
	ObligationID *int    `json:"obligation_id"`
	UserID       int     `json:"user_id"`
	Channel      string  `json:"channel"`
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	FCMToken     string  `json:"fcm_token"`
	ThreadTS     *string `json:"thread_ts"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackElement struct {
	Type     string    `json:"type"`
	Text     SlackText `json:"text"`
	ActionID string    `json:"action_id,omitempty"`
	Value    string    `json:"value,omitempty"`
	Style    string    `json:"style,omitempty"`
}

type SlackBlock struct {
	Type     string         `json:"type"`
	Text     *SlackText     `json:"text,omitempty"`
	Elements []SlackElement `json:"elements,omitempty"`
	BlockID  string         `json:"block_id,omitempty"`
}

type SlackMessage struct {
	Channel  string  `json:"channel"`
	ThreadTS *string `json:"thread_ts,omitempty"`
	Text     string  `json:"text"`
	Blocks   []Block `json:"blocks,omitempty"`
}

type SlackEvent struct {
	Token     string          `json:"token"`
	Challenge string          `json:"challenge,omitempty"`
	Type      string          `json:"type"`
	Event     json.RawMessage `json:"event,omitempty"`
}

type UpdatePayload struct {
	ViewID string     `json:"view_id"`
	View   UpdateView `json:"view"`
	Hash   string     `json:"hash"`
}

type UpdateView struct {
	CallbackID string            `json:"callback_id"`
	Type       string            `json:"type"`
	Title      SlackText         `json:"title"`
	Blocks     []json.RawMessage `json:"blocks"`
	Close      *SlackText        `json:"close,omitempty"`
	Submit     *SlackText        `json:"submit,omitempty"`
	Metadata   string            `json:"private_metadata,omitempty"`
}

type InnerEvent struct {
	Type     string `json:"type"`
	FileID   string `json:"file_id,omitempty"`
	ThreadTs string `json:"thread_ts,omitempty"`
	Ts       string `json:"ts,omitempty"`
	UserID   string `json:"user,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Text     string `json:"text,omitempty"`
}

type MessageEvent struct {
	Type     string      `json:"type"`
	Subtype  string      `json:"subtype,omitempty"`
	User     string      `json:"user"`
	Text     string      `json:"text"`
	Ts       string      `json:"ts"`
	Channel  string      `json:"channel"`
	Files    []SlackFile `json:"files,omitempty"`
	ThreadTs string      `json:"thread_ts,omitempty"`
	EventTs  string      `json:"event_ts,omitempty"`
}

type FileInfoRequestResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	File  struct {
		URLPrivate string `json:"url_private"`
		Name       string `json:"name"`
		MimeType   string `json:"mimetype"`
	} `json:"file"`
}

type SlackFile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Mimetype   string `json:"mimetype"`
	Filetype   string `json:"filetype"`
	URLPrivate string `json:"url_private"`
	Size       int64  `json:"size"`
	Title      string `json:"title"`
}

type ReceiveSlackFile struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type SlackMessageMeta struct {
	TS       string  `json:"ts"`
	ThreadTS *string `json:"thread_ts,omitempty"`
}

type SlackFilePayload struct {
	Files          []ReceiveSlackFile `json:"files"`
	InitialComment string             `json:"initial_comment,omitempty"`
	ChannelId      string             `json:"channel_id,omitempty"`
}

type SlackUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	TeamID   string `json:"team_id"`
}

type SlackTeam struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackAction struct {
	ActionID string    `json:"action_id"`
	BlockID  string    `json:"block_id"`
	Text     SlackText `json:"text"`
	Value    string    `json:"value"`
	Type     string    `json:"type"`
}

type StructuredModalValues struct {
	Channel      string  `json:"channel"`
	MessageId    *int    `json:"message_id"`
	ObligationId *int    `json:"obligation_id"`
	FcmToken     string  `json:"fcm_token"`
	UserId       int     `json:"user_id"`
	SlackUserId  string  `json:"slack_user_id"`
	TriggerId    *string `json:"trigger_id"`
	Text         *string
}

type SlackOption struct {
	Text  SlackText `json:"text"`
	Value string    `json:"value"`
}

type SlackValue struct {
	Type            string        `json:"type"`
	Value           string        `json:"value"`
	SelectedOption  *SlackOption  `json:"selected_option,omitempty"`
	SelectedOptions []SlackOption `json:"selected_options,omitempty"`
}

type SlackState struct {
	Values map[string]map[string]SlackValue `json:"values"`
}

type SlackView struct {
	ID         string            `json:"id"`
	Hash       string            `json:"hash,omitempty"`
	TriggerID  string            `json:"trigger_id,omitempty"`
	CallbackID string            `json:"callback_id"`
	Type       string            `json:"type"`
	Title      SlackText         `json:"title"`
	State      SlackState        `json:"state"`
	Metadata   string            `json:"private_metadata,omitempty"`
	Submit     *SlackText        `json:"submit,omitempty"`
	Close      *SlackText        `json:"close,omitempty"`
	Blocks     []json.RawMessage `json:"blocks"`
}

type SlackInteractionPayload struct {
	Type        string           `json:"type"`
	User        SlackUser        `json:"user"`
	Team        SlackTeam        `json:"team"`
	Hash        string           `json:"hash,omitempty"`
	Channel     SlackChannel     `json:"channel"`
	Action      SlackAction      `json:"action"`
	Actions     []SlackAction    `json:"actions"`
	TriggerID   string           `json:"trigger_id"`
	ResponseURL string           `json:"response_url"`
	View        *SlackView       `json:"view,omitempty"`
	CallbackID  string           `json:"callback_id,omitempty"`
	Message     SlackMessageMeta `json:"message"`
}

type SlackSuggestionPayload struct {
	Type       string    `json:"type"`
	TriggerID  string    `json:"trigger_id"`
	ActionID   string    `json:"action_id"`
	BlockID    string    `json:"block_id"`
	Value      string    `json:"value"`
	CallbackID string    `json:"callback_id"`
	User       SlackUser `json:"user"`
	Team       SlackTeam `json:"team"`
}

type SlackExtUrlResponse struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
}

type OptionGroup struct {
	Label   SlackText     `json:"label"`
	Options []SlackOption `json:"options"`
}

type SlackRequestResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Channel string `json:"channel,omitempty"`
	TS      string `json:"ts,omitempty"`
	Message any    `json:"message,omitempty"`
}

type SlackFileUploadResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	File  struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Name       string `json:"name"`
		Mimetype   string `json:"mimetype"`
		Size       int    `json:"size"`
		URLPrivate string `json:"url_private"`
		Shares     struct {
			Public map[string][]struct {
				TS        string `json:"ts"`
				Channel   string `json:"channel_name,omitempty"`
				User      string `json:"user,omitempty"`
				Permalink string `json:"permalink"`
			} `json:"public"`
			Private map[string][]struct {
				TS        string `json:"ts"`
				Channel   string `json:"channel_name,omitempty"`
				User      string `json:"user,omitempty"`
				Permalink string `json:"permalink"`
			} `json:"private"`
		} `json:"shares"`
	} `json:"file"`
}

type ModalInputElement struct {
	Type     string `json:"type"`
	ActionID string `json:"action_id"`

	Multiline    bool      `json:"multiline,omitempty"`
	MinLength    int       `json:"min_length,omitempty"`
	MaxLength    int       `json:"max_length,omitempty"`
	Placeholder  SlackText `json:"placeholder"`
	InitialValue string    `json:"initial_value,omitempty"`

	Options []SlackOption `json:"options,omitempty"`
}

type RadioButtonElement struct {
	Type     string        `json:"type"`
	ActionID string        `json:"action_id"`
	Options  []SlackOption `json:"options"`
}

type ModalActionElement struct {
	Type     string    `json:"type"`
	ActionID string    `json:"action_id"`
	Text     SlackText `json:"text"`
	Value    string    `json:"value"`
	Style    string    `json:"style,omitempty"`

	Options []SlackOption `json:"options,omitempty"`
}

type ActionBlock struct {
	Type     string               `json:"type"`
	BlockID  string               `json:"block_id"`
	Elements []ModalActionElement `json:"elements"`
}

type InputBlock struct {
	Type     string            `json:"type"`
	BlockID  string            `json:"block_id"`
	Element  ModalInputElement `json:"element"`
	Label    SlackText         `json:"label"`
	Optional bool              `json:"optional,omitempty"`
	Hint     *SlackText        `json:"hint,omitempty"`
}

type SectionAccessory struct {
	Radio  *RadioButtonElement
	Input  *ModalInputElement
	Action *ModalActionElement
}

type SectionBlock struct {
	Type      string              `json:"type"`
	BlockID   string              `json:"block_id,omitempty"`
	Text      *SlackText          `json:"text,omitempty"`
	Fields    []SlackText         `json:"fields,omitempty"`
	Accessory *RadioButtonElement `json:"accessory,omitempty"`
}

type Block interface {
	GetType() string
	GetBlockID() string
}

func (b SectionBlock) GetType() string {
	return b.Type
}

func (b SectionBlock) GetBlockID() string {
	return b.BlockID
}

func (b InputBlock) GetType() string {
	return b.Type
}

func (b InputBlock) GetBlockID() string {
	return b.BlockID
}

func (b ActionBlock) GetType() string {
	return b.Type
}

func (b ActionBlock) GetBlockID() string {
	return b.BlockID
}

type SlackModalView struct {
	Type          string     `json:"type"`
	CallbackID    string     `json:"callback_id"`
	Title         SlackText  `json:"title"`
	Submit        *SlackText `json:"submit,omitempty"`
	Close         *SlackText `json:"close,omitempty"`
	NotifyClosing bool       `json:"notify_on_close,omitempty"`
	Metadata      string     `json:"private_metadata,omitempty"`
	Blocks        []Block    `json:"blocks"`
}

type SlackModalPayload struct {
	TriggerID string         `json:"trigger_id"`
	View      SlackModalView `json:"view"`
}

type BlockSuggestionsResponse struct {
	Options []SlackOption `json:"options"`
}

type ModalValues struct {
	SelectedStatus      string
	EmploymentStatus    string
	SelectedGroups      []string
	SelectedGroup       string
	MessageType         string
	Translations        Translations
	UserID              string
	Obligations         []string
	DataType            string
	NewValue            string
	SelectedCountries   []string
	SelectedWorkpermits []string
	SelectedJobs        []string
	DocumentType        string
	Uploadable          bool
	TextField           bool
	StructuredMetadata  StructuredModalValues
}

type MessageUpdatePayload struct {
	ReplaceOriginal bool    `json:"replace_original"`
	DeleteOriginal  bool    `json:"delete_original,omitempty"`
	Text            string  `json:"text,omitempty"`
	Blocks          []Block `json:"blocks,omitempty"`
}
