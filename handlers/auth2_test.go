package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockUtilsService struct {
	mock.Mock
}

func (m *MockUtilsService) GetRequestSummary(r *http.Request) string {
	args := m.Called(r)
	return args.String(0)
}

func (m *MockUtilsService) ValidateStructAndRespond(req interface{}, w http.ResponseWriter, r *http.Request, requestSummary string, start time.Time) bool {
	args := m.Called(req, w, r, requestSummary, start)
	return args.Bool(0)
}

func (m *MockUtilsService) RespondWithError(w http.ResponseWriter, options utils.ErrorJSONResponseOptions) {
	m.Called(w, options)
}

func (m *MockUtilsService) RespondWithJSON(w http.ResponseWriter, options utils.SuccessJSONResponseOptions) {
	m.Called(w, options)
}

func (m *MockUtilsService) IsValidKenyanPhone(phone string) bool {
	args := m.Called(phone)
	return args.Bool(0)
}

func (m *MockUtilsService) GenerateOTP() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockUtilsService) GetCurrentFuncName() string {
	args := m.Called()
	return args.String(0)
}

type MockModelsService struct {
	mock.Mock
}

func (m *MockModelsService) CreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dtos.User), args.Error(1)
}

// Mock database for the test
func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	// Replace the global DB with our mock
	originalDB := models.DB
	models.DB = db

	return mock, func() {
		models.DB = originalDB
		db.Close()
	}
}

func TestRegisterHandler(t *testing.T) {
	mockUtils := new(MockUtilsService)

	// Setup database mock
	dbMock, cleanup := setupMockDB(t)
	defer cleanup()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Success with email",
			requestBody: dtos.RegisterRequest{
				Email:     "test@example.com",
				Firstname: "John",
				Lastname:  "Doe",
			},
			setupMocks: func() {
				// Mock database calls for CheckUserExistsByEmailOrPhone
				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE email = ?").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE phone_number = ?").
					WithArgs("").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Mock utils calls
				mockUtils.On("GetRequestSummary", mock.Anything).Return("request_summary")
				mockUtils.On("ValidateStructAndRespond", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true)
				mockUtils.On("IsValidKenyanPhone", mock.Anything).Maybe().Return(true)
				mockUtils.On("GenerateOTP").Return("123456", nil)
				mockUtils.On("GetCurrentFuncName").Return("RegisterHandler")
				mockUtils.On("RespondWithJSON", mock.Anything, mock.MatchedBy(func(opts utils.SuccessJSONResponseOptions) bool {
					return opts.CollectiveInfo.Code == http.StatusCreated
				})).Once()

				// Mock models.CreateUser database calls
				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE email = ?").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				dbMock.ExpectExec("INSERT INTO users").
					WithArgs(sqlmock.AnyArg(), "customer", "John", "Doe", "test@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid request - no email or phone",
			requestBody: dtos.RegisterRequest{
				Firstname: "John",
				Lastname:  "Doe",
			},
			setupMocks: func() {
				mockUtils.On("GetRequestSummary", mock.Anything).Return("request_summary")
				mockUtils.On("ValidateStructAndRespond", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true)
				mockUtils.On("GetCurrentFuncName").Return("RegisterHandler")
				mockUtils.On("RespondWithError", mock.Anything, mock.MatchedBy(func(opts utils.ErrorJSONResponseOptions) bool {
					return opts.CollectiveInfo.Code == http.StatusBadRequest && strings.Contains(opts.Message, "Phone number or email is required")
				})).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "User already exists by email",
			requestBody: dtos.RegisterRequest{
				Email:     "exist@example.com",
				Firstname: "John",
			},
			setupMocks: func() {
				// Mock database calls for CheckUserExistsByEmailOrPhone
				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE email = ?").
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				mockUtils.On("GetRequestSummary", mock.Anything).Return("request_summary")
				mockUtils.On("ValidateStructAndRespond", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true)
				mockUtils.On("GetCurrentFuncName").Return("RegisterHandler")
				mockUtils.On("RespondWithError", mock.Anything, mock.MatchedBy(func(opts utils.ErrorJSONResponseOptions) bool {
					return opts.CollectiveInfo.Code == http.StatusConflict && strings.Contains(opts.Message, "already exists")
				})).Once()
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUtils.ExpectedCalls = nil
			tt.setupMocks()

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			RegisterHandler(w, req)

			// Verify mocks
			mockUtils.AssertExpectations(t)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// Test individual components separately
func TestCheckUserExistsByEmailOrPhone_Unit(t *testing.T) {
	dbMock, cleanup := setupMockDB(t)
	defer cleanup()

	tests := []struct {
		name           string
		request        dtos.RegisterRequest
		setupMocks     func()
		expectedResult bool
	}{
		{
			name: "User exists by email",
			request: dtos.RegisterRequest{
				Email: "existing@example.com",
			},
			setupMocks: func() {
				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE email = ?").
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedResult: false, // Should return false (user exists)
		},
		{
			name: "User does not exist",
			request: dtos.RegisterRequest{
				Email: "new@example.com",
			},
			setupMocks: func() {
				dbMock.ExpectQuery("SELECT COUNT.*FROM users WHERE email = ?").
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedResult: true, // Should return true (user doesn't exist)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/register", nil)
			start := time.Now()

			result := CheckUserExistsByEmailOrPhone(w, r, tt.request, start, "request_summary")

			assert.Equal(t, tt.expectedResult, result)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestStoreOTPInRedis_Unit(t *testing.T) {
	redisClient, mock := redismock.NewClientMock()
	originalRedis := Redis
	Redis = redisClient
	defer func() {
		Redis = originalRedis
	}()

	tests := []struct {
		name        string
		userID      string
		otp         string
		ttl         time.Duration
		setupMock   func()
		expectError bool
	}{
		{
			name:   "Success",
			userID: "user123",
			otp:    "123456",
			ttl:    5 * time.Minute,
			setupMock: func() {
				mock.ExpectSetEX(fmt.Sprintf("otp:%s", "user123"), "123456", 5*time.Minute).SetVal("OK")
			},
			expectError: false,
		},
		{
			name:   "Redis error",
			userID: "user123",
			otp:    "123456",
			ttl:    5 * time.Minute,
			setupMock: func() {
				mock.ExpectSetEX(fmt.Sprintf("otp:%s", "user123"), "123456", 5*time.Minute).SetErr(fmt.Errorf("redis error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := StoreOTPInRedis(tt.userID, tt.otp, tt.ttl)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
