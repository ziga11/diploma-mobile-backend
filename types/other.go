package types

type BoolResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Reminder struct {
	MinDays      int
	MaxDays      int
	Translations Translations `json:"translations"`
}

type ZeroBounce struct {
	Address   string `json:"address"`
	Status    string `json:"status"`
	FreeEmail bool   `json:"free_email"`
	FirtName  string `json:"firstname"`
	LastName  string `json:"lastname"`
	Country   string `json:"country"`
	Gender    string `json:"gender"`
	Error     error  `json:"error"`
}

type BoardRequest struct {
	Type     string         `json:"type"`
	BoardID  int            `json:"board_id"`
	RowCount int            `json:"row_count"`
	Rows     map[string]Row `json:"rows"`
}

type Row struct {
	Field Field `json:"field"`
	Entry Entry `json:"entry"`
}

type Entry struct {
	ID    int     `json:"id"`
	Index int     `json:"index"`
	Value *string `json:"value"`
}

type Field struct {
	ID   int     `json:"id"`
	Name *string `json:"name"`
	Type string  `json:"type"`
}
