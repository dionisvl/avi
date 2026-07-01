package auth

type RegisterRequest struct {
	Email         string `json:"email" validate:"required,email"`
	Password      string `json:"password" validate:"required,min=8"`
	EmailVerified bool   `json:"email_verified"`
	Locale        string `json:"locale" validate:"omitempty,oneof=ru en"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"test@example.com" validate:"required,email"`
	Password string `json:"password" example:"password123" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type ResetPasswordRequestReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordConfirmReq struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type ResetPasswordSetReq struct {
	Email       string `json:"email" validate:"required,email"`
	Code        string `json:"code" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type ResendVerificationRequest struct {
	Email  string `json:"email" validate:"required,email"`
	Locale string `json:"locale" validate:"omitempty,oneof=ru en"`
}
