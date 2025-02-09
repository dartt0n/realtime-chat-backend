package models

// TokenDetails contains authentication token data including access and refresh tokens,
// their UUIDs and expiration timestamps
type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AccessUUID   string
	RefreshUUID  string
	AtExpires    int64
	RtExpires    int64
}

// AccessDetails contains the access token UUID and associated user ID
type AccessDetails struct {
	AccessUUID string
	UserID     string
}

// Token represents the JWT token pair returned to clients with access
// and refresh tokens
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
