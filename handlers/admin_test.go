package handlers

// import (
// 	"ekomasi_backend/dtos"
// 	"errors"
// 	"testing"
// )

// func TestCreateCategoryHandler(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    dtos.CreateCategory
// 		mockFunc func(dtos.CreateCategory) error
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: dtos.CreateCategory{Name: "Category 1", Description: "Description 1"},
// 			mockFunc: func(dtos.CreateCategory) error {
// 				return nil
// 			},
// 			wantErr: nil,
// 			wantMsg: "Category created successfully",
// 		},
// 		{
// 			name:  "No name and description",
// 			input: dtos.CreateCategory{},
// 			mockFunc: func(dtos.CreateCategory) error {
// 				return errors.New("Name and description is required")
// 			},
// 			wantErr: errors.New("Email or phone is required"),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, "", tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }

// func TestUpdateCategoryHandler(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    dtos.CreateCategory
// 		mockFunc func(dtos.CreateCategory) error
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: dtos.CreateCategory{Name: "Category 1", Description: "Description 1"},
// 			mockFunc: func(dtos.CreateCategory) error {
// 				return nil
// 			},
// 			wantErr: nil,
// 			wantMsg: "Category updated successfully",
// 		},
// 		{
// 			name:  "No name and description",
// 			input: dtos.CreateCategory{},
// 			mockFunc: func(dtos.CreateCategory) error {
// 				return errors.New("Name and description is required")
// 			},
// 			wantErr: errors.New("Email or phone is required"),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  "Category not found",
// 			input: dtos.CreateCategory{},
// 			mockFunc: func(dtos.CreateCategory) error {
// 				return errors.New("category not found")
// 			},
// 			wantErr: errors.New("Category with given ID not found"),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, "", tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }

// func TestDeleteCategoryHandler(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    map[string]any
// 		mockFunc func(map[string]any) error
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: nil,
// 			mockFunc: func(map[string]any) error {
// 				return nil
// 			},
// 			wantErr: nil,
// 			wantMsg: "Category deleted successfully",
// 		},
// 		{
// 			name:  "Category not found",
// 			input: nil,
// 			mockFunc: func(map[string]any) error {
// 				return errors.New("category not found")
// 			},
// 			wantErr: errors.New("Category with given ID not found"),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, "", tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }

// func TestCreateProductHandler(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    dtos.CreateProduct
// 		mockFunc func(dtos.CreateProduct) error
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: dtos.CreateProduct{Name: "Product 1", Description: "Description 1", SKU: "SKU1", Price: 1, CategoryID: "Category 1", StockQuantity: 10, SearchVector: "Product 1"},
// 			mockFunc: func(dtos.CreateProduct) error {
// 				return nil
// 			},
// 			wantErr: nil,
// 			wantMsg: "Product created successfully",
// 		},
// 		{
// 			name:  "Empty fields",
// 			input: dtos.CreateProduct{},
// 			mockFunc: func(dtos.CreateProduct) error {
// 				return errors.New("Name, description, SKU, price, category, stock quantity and search vector are required")
// 			},
// 			wantErr: errors.New("Email or phone is required"),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  "Category not found",
// 			input: dtos.CreateProduct{},
// 			mockFunc: func(dtos.CreateProduct) error {
// 				return errors.New("Category not found")
// 			},
// 			wantErr: errors.New("Category not found"),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  "Mapping to parent category",
// 			input: dtos.CreateProduct{},
// 			mockFunc: func(dtos.CreateProduct) error {
// 				return errors.New("Cannot map product to a parent category")
// 			},
// 			wantErr: errors.New("Cannot map product to a parent category"),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, "", tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }
