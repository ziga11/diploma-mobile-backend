package types

import (
	"encoding/json"
	"fmt"
)

type AccLangReq struct {
	AccID    int    `json:"account_id"`
	LangCode string `json:"lang_code"`
}

type Country struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type LangContent struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Translations struct {
	Si LangContent `json:"si"`
	En LangContent `json:"en"`
	Bs LangContent `json:"bs"`
}

func (t *Translations) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("translations: expected []byte, got %T", src)
	}

	return json.Unmarshal(b, t)
}
