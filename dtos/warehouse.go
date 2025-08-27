package dtos

type Warehouse struct {
	WarehouseID      string `json:"warehouse_id"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	WarehouseDetails string `json:"warehouse_details"`
}

type CreateWarehouseRequest struct {
	Name             string  `json:"name" validate:"required"`
	Location         string  `json:"location" validate:"required"`
	WarehouseDetails *string `json:"warehouse_details"`
}

type UpdateWarehouseRequest struct {
	Name             string `json:"name" validate:"required"`
	Location         string `json:"location" validate:"required"`
	WarehouseDetails string `json:"warehouse_details"`
}

type ListWarehousesResponse struct {
	Data []Warehouse    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}
