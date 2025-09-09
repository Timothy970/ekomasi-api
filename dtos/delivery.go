package dtos

type Delivery struct {
	DeliveryID      string `json:"delivery_id"`
	OrderID         string `json:"order_id" validate:"required"`
	DeliveryCharge  string `json:"delivery_charge" validate:"required"`
	Status          string `json:"status"`
	CourierDetails  string `json:"courier_details" validate:"required"`
	DeliveryAddress string `json:"delivery_address" validate:"required"`
}
type PagedDeliveries struct {
	Data []Delivery     `json:"data"`
	Meta PaginationMeta `json:"pagination"`
}
type UpdateDelivery struct {
	Status string `json:"status" validate:"required"`
}
