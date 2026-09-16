package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
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
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock)
		expectedStatus int
	}{
		{
			name: "Success with email",
			requestBody: dtos.RegisterRequest{
				Email:     "test@example.com",
				Firstname: "John",
				Lastname:  "Doe",
			},
			setupMocks: func(dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				// Mock database calls for CheckUserExistsByEmailOrPhone
				dbMock.ExpectQuery("SELECT u.user_id.* FROM users u").
					WithArgs("test@example.com", 1).
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "email", "name", "role_id", "phone_number", "status"}))

				// Expect Redis store for temporary registration data
				expectedData, _ := json.Marshal(map[string]interface{}{
					"email":     "test@example.com",
					"phone":     "",
					"firstname": "John",
					"lastname":  "Doe",
					"password":  "",
					"otp":       "2025",
				})
				redisMock.ExpectSet("pending_signup:test@example.com", expectedData, 10*time.Minute).SetVal("OK")

				// Expect Log Insert (triggered by RespondWithJSON in RegisterHandler)
				dbMock.ExpectExec("INSERT INTO logs").
					WithArgs(sqlmock.AnyArg(), "INFO", "Verify using the OTP sent within 10 minutes to complete registration.", sqlmock.AnyArg(), sqlmock.AnyArg(), "Auth", sqlmock.AnyArg()).
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
			setupMocks: func(dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				// Expect Log Insert (triggered by RespondWithError)
				dbMock.ExpectExec("INSERT INTO logs").
					WithArgs(sqlmock.AnyArg(), "WARN", "Phone number or email is required", sqlmock.AnyArg(), sqlmock.AnyArg(), "Auth", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "User already exists by email",
			requestBody: dtos.RegisterRequest{
				Email:     "existing@example.com",
				Firstname: "John",
			},
			setupMocks: func(dbMock sqlmock.Sqlmock, redisMock redismock.ClientMock) {
				// Mock database calls for CheckUserExistsByEmailOrPhone
				userRows := sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "email", "name", "role_id", "phone_number", "status"}).
					AddRow("user-1", "Existing", "User", "existing@example.com", "customer", "role-1", "123456", "active")
				dbMock.ExpectQuery("SELECT u.user_id.* FROM users u").
					WithArgs("existing@example.com", 1).
					WillReturnRows(userRows)
				dbMock.ExpectQuery("SELECT pm.permission_key from permissions_master pm").
					WithArgs("role-1").
					WillReturnRows(sqlmock.NewRows([]string{"permission_key"}))

				// Expect Log Insert (triggered by RespondWithError in CheckUserExistsByEmailOrPhone)
				dbMock.ExpectExec("INSERT INTO logs").
					WithArgs(sqlmock.AnyArg(), "WARN", "User with this email already exists", sqlmock.AnyArg(), sqlmock.AnyArg(), "Auth", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup database mock inside subtest
			dbMock, cleanup := setupMockDB(t)
			defer cleanup()

			// Setup Redis mock inside subtest
			redisClient, redisMock := redismock.NewClientMock()
			originalRedis := Redis
			Redis = redisClient
			defer func() {
				Redis = originalRedis
			}()

			tt.setupMocks(dbMock, redisMock)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			RegisterHandler(c)

			// Verify status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify mocks
			assert.NoError(t, dbMock.ExpectationsWereMet())
			assert.NoError(t, redisMock.ExpectationsWereMet())
		})
	}
}

// Test individual components separately
func TestCheckUserExistsByEmailOrPhone_Unit(t *testing.T) {
	tests := []struct {
		name           string
		request        dtos.RegisterRequest
		setupMocks     func(dbMock sqlmock.Sqlmock)
		expectedResult bool
	}{
		{
			name: "User exists by email",
			request: dtos.RegisterRequest{
				Email: "existing@example.com",
			},
			setupMocks: func(dbMock sqlmock.Sqlmock) {
				userRows := sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "email", "name", "role_id", "phone_number", "status"}).
					AddRow("user-1", "Existing", "User", "existing@example.com", "customer", "role-1", "123456", "active")
				dbMock.ExpectQuery("SELECT u.user_id.* FROM users u").
					WithArgs("existing@example.com", 1).
					WillReturnRows(userRows)
				dbMock.ExpectQuery("SELECT pm.permission_key from permissions_master pm").
					WithArgs("role-1").
					WillReturnRows(sqlmock.NewRows([]string{"permission_key"}))
				dbMock.ExpectExec("INSERT INTO logs").
					WithArgs(sqlmock.AnyArg(), "WARN", "User with this email already exists", sqlmock.AnyArg(), sqlmock.AnyArg(), "Auth", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedResult: false, // Should return false (user exists)
		},
		{
			name: "User does not exist",
			request: dtos.RegisterRequest{
				Email: "new@example.com",
			},
			setupMocks: func(dbMock sqlmock.Sqlmock) {
				dbMock.ExpectQuery("SELECT u.user_id.* FROM users u").
					WithArgs("new@example.com", 1).
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "email", "name", "role_id", "phone_number", "status"}))
			},
			expectedResult: true, // Should return true (user doesn't exist)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock, cleanup := setupMockDB(t)
			defer cleanup()

			tt.setupMocks(dbMock)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/register", nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			start := time.Now()

			result := CheckUserExistsByEmailOrPhone(c, tt.request, start, "request_summary")

			assert.Equal(t, tt.expectedResult, result)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestStoreOTPInRedis_Unit(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		otp         string
		ttl         time.Duration
		setupMock   func(redisMock redismock.ClientMock)
		expectError bool
	}{
		{
			name:   "Success",
			userID: "user123",
			otp:    "123456",
			ttl:    5 * time.Minute,
			setupMock: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet(fmt.Sprintf("otp:%s", "user123"), "123456", 5*time.Minute).SetVal("OK")
			},
			expectError: false,
		},
		{
			name:   "Redis error",
			userID: "user123",
			otp:    "123456",
			ttl:    5 * time.Minute,
			setupMock: func(redisMock redismock.ClientMock) {
				redisMock.ExpectSet(fmt.Sprintf("otp:%s", "user123"), "123456", 5*time.Minute).SetErr(fmt.Errorf("redis error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisClient, redisMock := redismock.NewClientMock()
			originalRedis := Redis
			Redis = redisClient
			defer func() {
				Redis = originalRedis
			}()

			tt.setupMock(redisMock)

			err := StoreOTPInRedis(tt.userID, tt.otp, tt.ttl)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, redisMock.ExpectationsWereMet())
		})
	}
}
