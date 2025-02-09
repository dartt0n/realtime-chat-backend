package forms

// Token represents a refresh token structure used for authentication and token renewal
type Token struct {
	RefreshToken string `form:"refresh_token" json:"refresh_token" binding:"required"`
}
