package handlers

import (
	"backend/types"
	"context"
	"fmt"
	"log"
	"net/http"
)

func ListOfCountries(ctx context.Context) []types.Country {
	tx, err := MobileDB.Begin()
	if err != nil {
		log.Println("Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	query := "SELECT id, name, code FROM mobile.country"

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		log.Println("Failed to list countries: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	var countries []types.Country

	for rows.Next() {
		var country types.Country
		if err := rows.Scan(&country.ID, &country.Name, &country.Code); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil
		}

		countries = append(countries, country)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating database rows: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed to commit the transaction %v --> ListOfCountries", err)
		return nil
	}

	return countries
}

func UserCountry(userId int) (types.Country, error) {
	query := `SELECT
			id, name
		  FROM mobile.country c
		  LEFT JOIN mobile.users u ON
			c.id = u.country_id
		  WHERE u.id = $1`

	var country types.Country

	err := MobileDB.QueryRow(query, userId).Scan(country.ID, country.Name)
	if err != nil {
		return types.Country{}, fmt.Errorf("Failed to find user company: %v", err)
	}

	return country, nil
}
