package handlers

import (
	"backend/types"
	"fmt"
	"log"
)

func CompanyByName(name string) (types.Company, error) {
	query := `SELECT id, name FROM mobile.company
			WHERE LOWER(name) = LOWER($1)`

	var company types.Company
	err := MobileDB.QueryRow(query, name).Scan(&company.ID, &company.Name)
	if err != nil {
		log.Printf("Failed to select company from db: %v", err)
		return types.Company{}, fmt.Errorf("Podjetja ni v podatkovni bazi")
	}

	return company, nil
}
