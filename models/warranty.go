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
	//check if the product already has a warranty/warranties
	// if there are there any, remove them first
	var warrantyIDS []string
	warrantyIDS, err = GetProductWarrantyIDs(pw.ProductID)
	if err != nil {
		return err
	}
	query := `INSERT INTO product_warranties (warranty_id, product_id, warranty_type_id, warranty_period, manufacturing_date, expiry_date) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, warrantyID, pw.ProductID, pw.WarrantyTypeID, pw.WarrantyPeriod, pw.ManufacturingDate, pw.ExpiryDate)
	if err != nil {
		return err
	}
	//if there are existing warranties, remove them
	for _, id := range warrantyIDS {
		err = RemoveProductWarranty(id)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetProductWarrantyIDs(productID string) ([]string, error) {
	query := `SELECT warranty_id FROM product_warranties WHERE product_id = ?`
	rows, err := DB.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var warrantyIDs []string
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		warrantyIDs = append(warrantyIDs, id)
	}
	return warrantyIDs, nil
}

func RemoveProductWarranty(productID string) error {
	//check if product exists
	err := IsProductThere(productID)
	if err != nil {
		return err
	}
	query := `DELETE FROM product_warranties WHERE product_id = ?`
	_, err = DB.Exec(query, productID)
	return err
}
