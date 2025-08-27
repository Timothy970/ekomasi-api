package dtos

// User represents a user account.
// type User struct {
// 	Username string  `json:"username"`
// 	Password string  `json:"password"`
// 	Email    string  `json:"email"`
// 	Role     string  `json:"role"`
// 	Profile  Profile `json:"profile"`
// }

// Profile represents user profile information.
type Profile struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	// Add more profile fields here
}

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Firstname   string `json:"first_name"`
	Lastname    string `json:"last_name"`
	Password    string `json:"password"`
	Email       string `json:"email" validate:"omitempty,required,email"`
	Phonenumber string `json:"phone_number"`
	Role        string `json:"role"`
}
type UserAdress struct {
	Address string `json:"address" validate:"required"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone_number"`
}

// UpdateProfileRequest represents the request body for updating user profile.
type UpdateProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"` // Add more profile fields here
}

// ForgotPasswordRequest represents the request body for forgot password.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents the request body for reset password.
type ResetPasswordRequest struct {
	Password string `json:"password"`
	Token    string `json:"token"`
}
type RegisterResponse struct {
	Message string                 `json:"message"`
	User    interface{}            `json:"user"` // Replace `interface{}` with your actual user struct if available
	Token   map[string]interface{} `json:"token"`
}
type UserAddress struct {
	AddressID string `json:"address_id"`
	Address   string `json:"address"`
}
