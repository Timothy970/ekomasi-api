package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractWSKey(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Explicit key parameter",
			url:      "/ws?key=custom_key_123",
			expected: "custom_key_123",
		},
		{
			name:     "Order ID and Delivery ID fallback",
			url:      "/ws?order_id=ord_1&delivery_id=del_2",
			expected: "ord_1:del_2",
		},
		{
			name:     "User ID fallback",
			url:      "/ws?user_id=usr_55",
			expected: "usr_55",
		},
		{
			name:     "Key parameter takes priority over user_id",
			url:      "/ws?key=priority_key&user_id=usr_55",
			expected: "priority_key",
		},
		{
			name:     "Missing parameters",
			url:      "/ws",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			actual := extractWSKey(req)
			if actual != tt.expected {
				t.Errorf("extractWSKey(%s) = %q; want %q", tt.url, actual, tt.expected)
			}
		})
	}
}

func TestHandleWebSocket_MissingKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()

	HandleWebSocket(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d for missing key, got %d", http.StatusBadRequest, w.Code)
	}
}
