package types

type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UserGroup struct {
	UserID    *int   `json:"user_id"`
	GroupID   *int   `json:"group_id"`
	GroupName string `json:"name"`
}
