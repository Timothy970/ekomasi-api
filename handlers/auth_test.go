package handlers

// import (
// 	"ekomasi_backend/dtos"
// 	"errors"
// 	"testing"
// )

// // --- Sentinel errors ---
// var (
// 	ErrSimulatedDB       = errors.New("simulated database connection failure")
// 	ErrInvalidEmail      = errors.New("invalid email format")
// 	ErrInvalidPhone      = errors.New("invalid phone number")
// 	ErrUserAlreadyExists = errors.New("user already exists")
// 	PhoneNumber          = "254712345678"
// 	VerificationFailed   = "Verification unsuccessful"
// 	NoUserFound          = "No user Found"
// 	EmailorPhone         = "Email or phone required"
// 	EmailPhone           = "No Email and Phone"
// )

// // --- Mock Implementations ---
// func mockSuccessCreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
// 	return &dtos.User{
// 		ID:        "mock-user-id-123",
// 		FirstName: input.Firstname,
// 		LastName:  input.Lastname,
// 		Email:     input.Email,
// 		Role:      "customer",
// 	}, nil
// }
// func mockFailureCreateUser(input dtos.RegisterRequest) (*dtos.User, error) {
// 	return nil, ErrSimulatedDB
// }
// func mockInvalidEmail(input dtos.RegisterRequest) (*dtos.User, error) {
// 	return nil, ErrInvalidEmail
// }
// func mockInvalidPhone(input dtos.RegisterRequest) (*dtos.User, error) {
// 	return nil, ErrInvalidPhone
// }
// func mockUserExists(input dtos.RegisterRequest) (*dtos.User, error) {
// 	return nil, ErrUserAlreadyExists
// }

// // --- Table-driven Unit Test ---
// func TestCreateUser(t *testing.T) {
// 	testCases := []struct {
// 		name      string
// 		input     dtos.RegisterRequest
// 		mockFunc  func(dtos.RegisterRequest) (*dtos.User, error)
// 		wantErr   error
// 		wantEmail string
// 	}{
// 		{
// 			name: "success",
// 			input: dtos.RegisterRequest{
// 				Email:       "test@example.com",
// 				Phonenumber: PhoneNumber,
// 			},
// 			mockFunc:  mockSuccessCreateUser,
// 			wantErr:   nil,
// 			wantEmail: "test@example.com",
// 		},
// 		{
// 			name: "db error",
// 			input: dtos.RegisterRequest{
// 				Email:       "fail@example.com",
// 				Phonenumber: PhoneNumber,
// 			},
// 			mockFunc: mockFailureCreateUser,
// 			wantErr:  ErrSimulatedDB,
// 		},
// 		{
// 			name: "invalid email",
// 			input: dtos.RegisterRequest{
// 				Email:       "not-an-email",
// 				Phonenumber: PhoneNumber,
// 			},
// 			mockFunc: mockInvalidEmail,
// 			wantErr:  ErrInvalidEmail,
// 		},
// 		{
// 			name: "invalid phone",
// 			input: dtos.RegisterRequest{
// 				Firstname:   "Jane",
// 				Phonenumber: "12345",
// 			},
// 			mockFunc: mockInvalidPhone,
// 			wantErr:  ErrInvalidPhone,
// 		},
// 		{
// 			name: "user already exists",
// 			input: dtos.RegisterRequest{
// 				Email: "existing@example.com",
// 			},
// 			mockFunc: mockUserExists,
// 			wantErr:  ErrUserAlreadyExists,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			runUserTest(t, tc)
// 		})
// 	}

// }
// func runUserTest(t *testing.T, tc struct {
// 	name      string
// 	input     dtos.RegisterRequest
// 	mockFunc  func(dtos.RegisterRequest) (*dtos.User, error)
// 	wantErr   error
// 	wantEmail string
// }) {
// 	t.Helper()
// 	user, err := tc.mockFunc(tc.input)

// 	if tc.wantErr == nil {
// 		if err != nil {
// 			t.Fatalf("unexpected error: %v", err)
// 		}
// 		if user == nil {
// 			t.Fatal("expected user, got nil")
// 		}
// 		if user.Email != tc.wantEmail {
// 			t.Errorf("expected email %s, got %s", tc.wantEmail, user.Email)
// 		}
// 	} else {
// 		if user != nil {
// 			t.Fatalf("expected nil user, got %+v", user)
// 		}
// 		if !errors.Is(err, tc.wantErr) {
// 			t.Errorf("expected error %v, got %v", tc.wantErr, err)
// 		}
// 	}
// }

// // Mock function signature: returns (error, message string)
// type mockVerifyFunc func(dtos.VerifyOTP) (error, string)

// func assertVerifyResult(t *testing.T, gotErr error, gotMsg string, wantErr error, wantMsg string) {
// 	t.Helper()

// 	if wantErr == nil {
// 		if gotErr != nil {
// 			t.Fatalf("unexpected error: %v", gotErr)
// 		}
// 	} else {
// 		if gotErr == nil {
// 			t.Fatalf("expected error %v, got nil", wantErr)
// 		}
// 		if gotErr.Error() != wantErr.Error() {
// 			t.Errorf("expected error %v, got %v", wantErr, gotErr)
// 		}
// 	}

// 	if gotMsg != wantMsg {
// 		t.Errorf("expected msg %q, got %q", wantMsg, gotMsg)
// 	}
// }

// func TestVerifySign(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    dtos.VerifyOTP
// 		mockFunc mockVerifyFunc
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: dtos.VerifyOTP{Phone: PhoneNumber, OTP: "1234"},
// 			mockFunc: func(dtos.VerifyOTP) (error, string) {
// 				return nil, "Verification successful"
// 			},
// 			wantErr: nil,
// 			wantMsg: "Verification successful",
// 		},
// 		{
// 			name:  EmailPhone,
// 			input: dtos.VerifyOTP{OTP: "1234"},
// 			mockFunc: func(dtos.VerifyOTP) (error, string) {
// 				return errors.New(EmailorPhone), VerificationFailed
// 			},
// 			wantErr: errors.New(EmailorPhone),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  "No otp",
// 			input: dtos.VerifyOTP{Phone: PhoneNumber},
// 			mockFunc: func(dtos.VerifyOTP) (error, string) {
// 				return errors.New("Otp cannot be empty"), VerificationFailed
// 			},
// 			wantErr: errors.New("Otp cannot be empty"),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  NoUserFound,
// 			input: dtos.VerifyOTP{Phone: PhoneNumber, OTP: "1234"},
// 			mockFunc: func(dtos.VerifyOTP) (error, string) {
// 				return errors.New(NoUserFound), VerificationFailed
// 			},
// 			wantErr: errors.New(NoUserFound),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  "invalid otp",
// 			input: dtos.VerifyOTP{Phone: PhoneNumber, OTP: "123"},
// 			mockFunc: func(dtos.VerifyOTP) (error, string) {
// 				return errors.New("Invalid or expired otp"), VerificationFailed
// 			},
// 			wantErr: errors.New("Invalid or expired otp"),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err, msg := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, msg, tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }
// func TestUserLogin(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    map[string]any
// 		mockFunc func(map[string]any) error
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: map[string]any{"phone": PhoneNumber},
// 			mockFunc: func(map[string]any) error {
// 				return nil
// 			},
// 			wantErr: nil,
// 			wantMsg: "OTP sent",
// 		},
// 		{
// 			name:  EmailPhone,
// 			input: map[string]any{},
// 			mockFunc: func(map[string]any) error {
// 				return errors.New("Email or phone is required")
// 			},
// 			wantErr: errors.New("Email or phone is required"),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  NoUserFound,
// 			input: map[string]any{"phone": PhoneNumber},
// 			mockFunc: func(map[string]any) error {
// 				return errors.New(NoUserFound)
// 			},
// 			wantErr: errors.New(NoUserFound),
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

// func TestResendOTP(t *testing.T) {
// 	testCases := []struct {
// 		name     string
// 		input    dtos.ResendOTP
// 		mockFunc func(dtos.ResendOTP) (error, string)
// 		wantErr  error
// 		wantMsg  string
// 	}{
// 		{
// 			name:  "success",
// 			input: dtos.ResendOTP{Phone: PhoneNumber},
// 			mockFunc: func(dtos.ResendOTP) (error, string) {
// 				return nil, "Otp resent successfully"
// 			},
// 			wantErr: nil,
// 			wantMsg: "Otp resent",
// 		},
// 		{
// 			name:  EmailPhone,
// 			input: dtos.ResendOTP{},
// 			mockFunc: func(dtos.ResendOTP) (error, string) {
// 				return errors.New(EmailorPhone), VerificationFailed
// 			},
// 			wantErr: errors.New(EmailorPhone),
// 			wantMsg: VerificationFailed,
// 		},
// 		{
// 			name:  NoUserFound,
// 			input: dtos.ResendOTP{Phone: PhoneNumber},
// 			mockFunc: func(dtos.ResendOTP) (error, string) {
// 				return errors.New(NoUserFound), VerificationFailed
// 			},
// 			wantErr: errors.New(NoUserFound),
// 			wantMsg: VerificationFailed,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err, msg := tc.mockFunc(tc.input)
// 			assertVerifyResult(t, err, msg, tc.wantErr, tc.wantMsg)
// 		})
// 	}
// }
