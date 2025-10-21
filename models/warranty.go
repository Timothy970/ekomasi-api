package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"log"

	"github.com/teris-io/shortid"
)

func CreateWarrantType(wt dtos.CreateWarrantyTypeRequest) error {
	warrantyID, _ := shortid.Generate()
	query := `INSERT INTO warranty_types (warranty_type_id, name, description) VALUES (?, ?, ?)`
	_, err := DB.Exec(query, warrantyID, wt.Name, wt.Description)
	return err
}

func GetAllWarrantTypes() ([]dtos.WarrantyType, error) {
	query := `SELECT warranty_type_id, name, description FROM warranty_types`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var warranties []dtos.WarrantyType
	for rows.Next() {
		var wt dtos.WarrantyType

		err := rows.Scan(&wt.WarrantyTypeID, &wt.Name, &wt.Description)
		if err != nil {
			return nil, err
		}
		warranties = append(warranties, wt)
	}
	return warranties, nil
}
func isWarrantyTypeThere(warrantyID string) error {
	exists, err := RecordExists("warranty_types", "warranty_type_id = ?", warrantyID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("warranty type with ID %s does not exist", warrantyID)
	}
	return nil
}
func UpdateWarrantType(warrantyID string, wt dtos.WarrantyType) error {
	if err := isWarrantyTypeThere(warrantyID); err != nil {
		return err
	}
	query := `UPDATE warranty_types SET name = ?, description = ? WHERE warranty_type_id = ?`
	_, err := DB.Exec(query, wt.Name, wt.Description, warrantyID)
	return err
}

func DeleteWarrantType(warrantyID string) error {
	if err := isWarrantyTypeThere(warrantyID); err != nil {
		return err
	}
	query := `DELETE FROM warranty_types WHERE warranty_type_id = ?`
	_, err := DB.Exec(query, warrantyID)
	return err
}

func AddProductWarranties(pw dtos.AddProductWarrantiesRequest) error {
	//check if product exists
	err := IsProductThere(pw.ProductID)
	if err != nil {
		return err
	}
	//check if warranty type exists
	err = isWarrantyTypeThere(pw.WarrantyTypeID)
	if err != nil {
		log.Printf("warranty type check error: %s with id %s", err, pw.WarrantyTypeID)
		return err
	}
	warrantyID, _ := shortid.Generate()
	query := `INSERT INTO product_warranties (warranty_id, product_id, warranty_type_id, warranty_period, manufacturing_date, expiry_date) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, warrantyID, pw.ProductID, pw.WarrantyTypeID, pw.WarrantyPeriod, pw.ManufacturingDate, pw.ExpiryDate)
	return err
}
