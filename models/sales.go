package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"time"
)

var nowTime = "2006-01-02 15:04:05"

type SalesSummaryOverTime struct {
	Period struct {
		Start string  `json:"start"`
		End   string  `json:"end"`
		Type  *string `json:"type"`
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`
		Revenue          float64 `json:"revenue"`
		GrowthPercentage float64 `json:"growth_percentage"`
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"`
			Revenue     float64 `json:"revenue"`
		} `json:"previous_period"`
	} `json:"comparison"`
}
type SalesTrendsSummary struct {
	Period struct {
		StartPeriod struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"start_period"`
		EndPeriod struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"end_period"`
	} `json:"period"`
	Metrics struct {
		SalesVolume      int     `json:"sales_volume"`
		Revenue          float64 `json:"revenue"`
		GrowthPercentage float64 `json:"growth_percentage"`
	} `json:"metrics"`
	Comparison struct {
		PreviousPeriod struct {
			SalesVolume int     `json:"sales_volume"`
			Revenue     float64 `json:"revenue"`
		} `json:"previous_period"`
	} `json:"comparison"`
}

func GetSalesTrendsSummary(start, end, prevStart, prevEnd time.Time) (SalesTrendsSummary, error) {

	var currentVolume int
	var currentRevenue float64
	err := DB.QueryRow(`
		SELECT COALESCE(SUM(oi.quantity), 0),
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
	`, start, end).Scan(&currentVolume, &currentRevenue)
	if err != nil {
		return SalesTrendsSummary{}, err
	}

	var prevVolume int
	var prevRevenue float64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(oi.quantity), 0),
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0)
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
	`, prevStart, prevEnd).Scan(&prevVolume, &prevRevenue)
	if err != nil {
		return SalesTrendsSummary{}, err
	}

	growth := 0.0
	if prevRevenue > 0 {
		growth = ((currentRevenue - prevRevenue) / prevRevenue) * 100
	}

	response := SalesTrendsSummary{}
	response.Period.StartPeriod.Start = fmt.Sprintf("%s", start.Format(nowTime))
	response.Period.StartPeriod.End = fmt.Sprintf("%s", end.Format(nowTime))
	response.Period.EndPeriod.Start = fmt.Sprintf("%s", prevStart.Format(nowTime))
	response.Period.EndPeriod.End = fmt.Sprintf("%s", prevEnd.Format(nowTime))
	response.Metrics.SalesVolume = currentVolume
	response.Metrics.Revenue = currentRevenue
	response.Metrics.GrowthPercentage = growth
	response.Comparison.PreviousPeriod.SalesVolume = prevVolume
	response.Comparison.PreviousPeriod.Revenue = prevRevenue

	return response, nil
}

type SalesTrendPoint struct {
	Date             string  `json:"date"`
	SalesVolume      int     `json:"sales_volume"`
	Revenue          float64 `json:"revenue"`
	GrowthPercentage float64 `json:"growth_percentage"`
}

type SalesTrendsOverTime struct {
	Period struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Type  string `json:"type"`
	} `json:"period"`
	Data []SalesTrendPoint `json:"data"`
}

func GetSalesTrendsOverTime(period string, start, end time.Time) (SalesTrendsOverTime, error) {

	var groupBy string
	switch period {
	case "week":
		groupBy = "YEARWEEK(o.created_at)"
	case "month":
		groupBy = "DATE_FORMAT(o.created_at, '%Y-%m')"
	case "quarter":
		groupBy = "CONCAT(YEAR(o.created_at), '-Q', QUARTER(o.created_at))"
	case "year":
		groupBy = "YEAR(o.created_at)"
	default:
		groupBy = "DATE(o.created_at)"
	}

	query := fmt.Sprintf(`
		SELECT %s AS period_date,
		       COALESCE(SUM(oi.quantity), 0) AS sales_volume,
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS revenue
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
		GROUP BY period_date
		ORDER BY period_date ASC
	`, groupBy)

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return SalesTrendsOverTime{}, err
	}
	defer rows.Close()

	var data []SalesTrendPoint
	var prevRevenue float64

	for rows.Next() {
		var periodDate string
		var salesVolume int
		var revenue float64

		if err := rows.Scan(&periodDate, &salesVolume, &revenue); err != nil {
			return SalesTrendsOverTime{}, err
		}

		growth := 0.0
		if prevRevenue > 0 {
			growth = ((revenue - prevRevenue) / prevRevenue) * 100
		}
		prevRevenue = revenue

		data = append(data, SalesTrendPoint{
			Date:             periodDate,
			SalesVolume:      salesVolume,
			Revenue:          revenue,
			GrowthPercentage: growth,
		})
	}

	response := SalesTrendsOverTime{}
	response.Period.Start = fmt.Sprintf("%s", start.Format(nowTime))
	response.Period.End = fmt.Sprintf("%s", end.Format(nowTime))
	response.Period.Type = period
	response.Data = data

	return response, nil
}

// Customer segmentation by delivery location
func GetCustomerSegmentation(start, end time.Time) ([]dtos.CustomerSegment, error) {
	query := `
        SELECT 
            d.delivery_address AS segment_name,
            COUNT(DISTINCT o.order_id) AS order_count,
            SUM(oi.unit_price * oi.quantity) AS total_sales,
            AVG(oi.unit_price * oi.quantity) AS avg_purchase_value,
            (SUM(oi.unit_price * oi.quantity) / COUNT(DISTINCT o.order_id)) AS avg_order_value,
            COUNT(DISTINCT o.order_id) AS transactions
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY d.delivery_address
    `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []dtos.CustomerSegment
	for rows.Next() {
		var seg dtos.CustomerSegment
		if err := rows.Scan(&seg.SegmentName, &seg.OrderCount, &seg.TotalSales, &seg.AvgPurchaseValue, &seg.AvgOrderValue, &seg.Transactions); err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}
	return segments, nil
}

// Sales by region
func GetSalesByRegion(start, end time.Time) ([]dtos.RegionSales, error) {
	query := `
        SELECT 
            d.delivery_address AS region,
            SUM(oi.unit_price * oi.quantity) AS total_sales,
            (SUM(oi.unit_price * oi.quantity) / COUNT(DISTINCT o.order_id)) AS avg_order_value,
            COUNT(DISTINCT o.order_id) AS transactions
        FROM orders o
        JOIN order_items oi ON o.order_id = oi.order_id
        JOIN deliveries d ON o.delivery_id = d.delivery_id
        WHERE o.created_at BETWEEN ? AND ?
        GROUP BY d.delivery_address
    `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []dtos.RegionSales
	for rows.Next() {
		var rgn dtos.RegionSales
		if err := rows.Scan(&rgn.Region, &rgn.TotalSales, &rgn.AvgOrderValue, &rgn.Transactions); err != nil {
			return nil, err
		}
		regions = append(regions, rgn)
	}
	return regions, nil
}
