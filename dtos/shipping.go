package dtos

type ShippingCostRequest struct {
	Location string `json:"location"`
}

type ShippingCostResponse struct {
	Location string  `json:"location" validate:"required"`
	Charge   float64 `json:"charge"`
}
type Location struct {
	ID       uint64  `json:"id"`
	Location string  `json:"location"`
	Charge   float64 `json:"charge"`
}
type UpdateLocation struct {
	Charge   float64 `json:"charge" validate:"required"`
	Location string  `json:"location" validate:"required"`
}
type LocationID struct {
	ID int `json:"location_id"`
}
