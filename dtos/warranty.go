package dtos

type CreateWarrantyTypeRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
}

type WarrantyType struct {
	WarrantyTypeID string `json:"warranty_type_id"`
	Name           string `json:"name" validate:"required"`
	Description    string `json:"description" validate:"required"`
}

type AddProductWarrantiesRequest struct {
	ProductID         string `json:"product_id" validate:"required"`
	WarrantyTypeID    string `json:"warranty_type_id" validate:"required"`
	WarrantyPeriod    int    `json:"warranty_period" validate:"required,min=1"` // in months
	ManufacturingDate string `json:"manufacturing_date" validate:"required"`
	ExpiryDate        string `json:"expiry_date" validate:"required,gtfield=ManufacturingDate"`
}
