package dtos

type Supplier struct {
	SupplierID   string  `json:"supplier_id"`
	Name         string  `json:"name" validate:"required"`
	ContactEmail string  `json:"contact_email" validate:"email"`
	ContactPhone string  `json:"contact_phone" validate:"required"`
	ExtraDetails *string `json:"extra_details,omitempty"`
}
