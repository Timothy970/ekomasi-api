// Package utils provides utility functions for the Adenzo backend API.
// This file contains utilities for generating and managing account codes
// for the chart of accounts system.
package utils

import (
	"adenzo_backend/dtos"
	"errors"
	"fmt"
	"strings"
)

// GetAccountCodeRange returns the code range for a given account type
// Asset: 10000-19999
// Liability: 20000-29999
// Equity: 30000-39999
// Revenue: 40000-49999
// Expense: 50000-59999
func GetAccountCodeRange(accountType string) (dtos.AccountCodeRange, error) {
	accountTypeLower := strings.ToLower(accountType)

	switch accountTypeLower {
	case "asset":
		return dtos.AccountCodeRange{Min: 10000, Max: 19999}, nil
	case "liability":
		return dtos.AccountCodeRange{Min: 20000, Max: 29999}, nil
	case "equity":
		return dtos.AccountCodeRange{Min: 30000, Max: 39999}, nil
	case "revenue":
		return dtos.AccountCodeRange{Min: 40000, Max: 49999}, nil
	case "expense":
		return dtos.AccountCodeRange{Min: 50000, Max: 59999}, nil
	default:
		return dtos.AccountCodeRange{}, errors.New("invalid account type")
	}
}

// ValidateAccountCode checks if an account code is within the valid range for the account type
func ValidateAccountCode(accountCode string, accountType string) error {
	codeRange, err := GetAccountCodeRange(accountType)
	if err != nil {
		return err
	}

	// Convert code to integer
	var code int
	_, err = fmt.Sscanf(accountCode, "%d", &code)
	if err != nil {
		return errors.New("invalid account code format")
	}

	if code < codeRange.Min || code > codeRange.Max {
		return fmt.Errorf("account code must be between %d and %d for %s accounts",
			codeRange.Min, codeRange.Max, accountType)
	}

	return nil
}
