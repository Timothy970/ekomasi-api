package utils

import (
	"bytes"
	"strings"
	"testing"
)

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

func newMockFile(content string) *mockMultipartFile {
	return &mockMultipartFile{
		Reader: bytes.NewReader([]byte(content)),
	}
}

func TestParseProductsCSV_Valid(t *testing.T) {
	header := "name,description,sku,price,sub_category_id,stock_quantity,tag,low_stock_quantity_warning,barcode,buying_price,weight,weight_limit,dimensions,brand,manufacturer,warranty_period,combination_name,combination_sku,additional_price,option_1,option_2,option_3\n"
	row1 := "Product A,Desc A,SKU001,10.5,cat1,100,tag1,10,123456,5.0,1.5,10.0,10x10,BrandA,ManA,12,CombA,CSKU001,2.0,Opt1,Opt2,Opt3\n"
	csvContent := header + row1

	products, err := ParseProductsCSV(newMockFile(csvContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	p := products[0]
	if p.Name != "Product A" {
		t.Errorf("expected name 'Product A', got '%s'", p.Name)
	}
	if p.SKU != "SKU001" {
		t.Errorf("expected SKU 'SKU001', got '%s'", p.SKU)
	}
	if p.Price != 10.5 {
		t.Errorf("expected price 10.5, got %f", p.Price)
	}
}

func TestParseProductsCSV_MissingHeaderColumns(t *testing.T) {
	header := "name,description,sku\n"
	_, err := ParseProductsCSV(newMockFile(header))
	if err == nil {
		t.Fatal("expected error for missing headers, got nil")
	}
	if !strings.Contains(err.Error(), "invalid CSV: missing required columns") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseProductsCSV_MissingRequiredFieldInRow(t *testing.T) {
	header := "name,description,sku,price,sub_category_id,stock_quantity,tag,low_stock_quantity_warning,barcode,buying_price,weight,weight_limit,dimensions,brand,manufacturer,warranty_period,combination_name,combination_sku,additional_price,option_1,option_2,option_3\n"
	// Missing name field in row 1
	row1 := ",Desc A,SKU001,10.5,cat1,100,tag1,10,123456,5.0,1.5,10.0,10x10,BrandA,ManA,12,CombA,CSKU001,2.0,Opt1,Opt2,Opt3\n"
	csvContent := header + row1

	_, err := ParseProductsCSV(newMockFile(csvContent))
	if err == nil {
		t.Fatal("expected error when all rows skipped, got nil")
	}
	if err.Error() != "no valid products found in CSV" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseProductsCSV_EmptyRowsAndSkippedRows(t *testing.T) {
	header := "name,description,sku,price,sub_category_id,stock_quantity,tag,low_stock_quantity_warning,barcode,buying_price,weight,weight_limit,dimensions,brand,manufacturer,warranty_period,combination_name,combination_sku,additional_price,option_1,option_2,option_3\n"
	emptyRow := "  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  ,  \n"
	invalidRow := ",Desc A,SKU001,10.5,cat1,100,tag1,10,123456,5.0,1.5,10.0,10x10,BrandA,ManA,12,CombA,CSKU001,2.0,Opt1,Opt2,Opt3\n"
	validRow := "Product B,Desc B,SKU002,20.0,cat2,50,tag2,5,654321,10.0,2.0,15.0,20x20,BrandB,ManB,6,CombB,CSKU002,3.0,Opt1,Opt2,Opt3\n"

	csvContent := header + emptyRow + invalidRow + validRow

	products, err := ParseProductsCSV(newMockFile(csvContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].Name != "Product B" {
		t.Errorf("expected product name 'Product B', got '%s'", products[0].Name)
	}
}
