package slack

import (
	"backend/handlers"
	"backend/types"
	"fmt"
)

func userSelectSuggestions(interaction types.SlackSuggestionPayload, response *types.BlockSuggestionsResponse) error {
	query := interaction.Value
	users, err := handlers.SearchUsersInDB(query)
	if err != nil {
		return fmt.Errorf("Failed to find a user")
	}

	options := []types.SlackOption{}
	for _, u := range users {
		options = append(options, types.SlackOption{
			Text: types.SlackText{
				Type: "plain_text",
				Text: fmt.Sprintf("%s %s", u.FirstName, u.LastName),
			},
			Value: fmt.Sprintf("%d", u.ID),
		})
	}

	*response = types.BlockSuggestionsResponse{Options: options}

	return nil
}
