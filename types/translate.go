package types

type TranslateReq struct {
	Text   []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
}

type Translation struct {
	Text string `json:"translatedText"`
}

type TranslationResponse struct {
	Data struct {
		Translations []Translation `json:"translations"`
	} `json:"data"`
}
