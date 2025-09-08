package dtos

import "time"

type CustomerSegmentationResponse struct {
	Period struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"period"`
	Segments []CustomerSegment `json:"segments"`
}

type CustomerSegment struct {
	SegmentName      string  `json:"segment_name"`
	OrderCount       int     `json:"order_count"`
	TotalSales       float64 `json:"total_sales"`
	AvgPurchaseValue float64 `json:"avg_purchase_value"`
	AvgOrderValue    float64 `json:"avg_order_value"`
	Transactions     int     `json:"transactions"`
}

type SalesByRegionResponse struct {
	Period struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"period"`
	Regions []RegionSales `json:"regions"`
}

type RegionSales struct {
	Region        string  `json:"region"`
	TotalSales    float64 `json:"total_sales"`
	AvgOrderValue float64 `json:"avg_order_value"`
	Transactions  int     `json:"transactions"`
}
