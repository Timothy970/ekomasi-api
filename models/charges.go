package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"

	"github.com/teris-io/shortid"
)

func isChargeThere(id string) error {
	exists, err := RecordExists("charges", "charge_id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("charge not found")
	}
	return nil
}
func AddCharge(input dtos.Charge) (*dtos.Charge, error) {
	chargeID, _ := shortid.Generate()
	_, err := DB.Exec(`
		INSERT INTO charges (charge_id, charge_name, charge_value)
		VALUES (?, ?, ?)`,
		chargeID, input.Type, input.Value,
	)
	if err != nil {
		return nil, err
	}
	return &dtos.Charge{
		ID:    chargeID,
		Type:  input.Type,
		Value: input.Value,
	}, nil
}

func UpdateCharge(id string, input dtos.Charge) (*dtos.Charge, error) {
	err := isChargeThere(id)
	if err != nil {
		return nil, err
	}
	_, err = DB.Exec(`
		UPDATE charges
		SET charge_name = ?, charge_value = ?
		WHERE charge_id = ?`,
		input.Type, input.Value, id,
	)
	if err != nil {
		return nil, err
	}
	return &dtos.Charge{
		ID:    id,
		Type:  input.Type,
		Value: input.Value,
	}, nil
}

func GetChargeByID(id string) (*dtos.Charge, error) {
	err := isChargeThere(id)
	if err != nil {
		return nil, err
	}
	row := DB.QueryRow(`SELECT charge_id, charge_name, charge_value FROM charges WHERE charge_id = ?`, id)

	var c dtos.Charge
	if err := row.Scan(&c.ID, &c.Type, &c.Value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func GetAllCharges() ([]dtos.Charge, error) {
	rows, err := DB.Query(`SELECT charge_id, charge_name, charge_value FROM charges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charges []dtos.Charge
	for rows.Next() {
		var c dtos.Charge
		if err := rows.Scan(&c.ID, &c.Type, &c.Value); err != nil {
			return nil, err
		}
		charges = append(charges, c)
	}
	return charges, nil
}

func DeleteCharge(id string) error {
	err := isChargeThere(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM charges WHERE charge_id = ?`, id)
	return err
}

func AddChargeToProduct(input dtos.AddChargeToProductRequest) error {
	//check if product exists
	err := isProductThere(input.ProductID)
	if err != nil {
		return err
	}
	//check if charge exists
	err = isChargeThere(input.ChargeID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		INSERT INTO product_charges (product_id, charge_id)
		VALUES (?, ?)`,
		input.ProductID, input.ChargeID,
	)
	return err
}
