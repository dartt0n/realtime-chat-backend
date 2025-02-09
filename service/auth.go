package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/dartt0n/realtime-chat-backend/models"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// AuthService handles authentication related operations using a key-value store
type AuthService struct {
	kv kv.KeyValueStore
}

// NewAuthService creates a new AuthService instance with the provided key-value store
func NewAuthService(kv kv.KeyValueStore) *AuthService {
	return &AuthService{
		kv: kv,
	}
}

// CreateToken generates access and refresh tokens for a given user ID
func (s AuthService) CreateToken(userID models.UserID) (*models.TokenDetails, error) {
	td := &models.TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix() // 15 minutes
	td.AccessUUID = uuid.New().String()

	td.RtExpires = time.Now().Add(time.Hour * 24 * 3).Unix() // 3 days
	td.RefreshUUID = uuid.New().String()

	var err error
	//Creating Access Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUUID
	atClaims["user_id"] = userID
	atClaims["exp"] = td.AtExpires

	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		slog.Error("failed to create access token", "error", err, "user_id", userID)
		return nil, err
	}

	//Creating Refresh Token
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUUID
	rtClaims["user_id"] = userID.String()
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		slog.Error("failed to create refresh token", "error", err, "user_id", userID)
		return nil, err
	}
	return td, nil
}

// CreateAuth stores the token details in the key-value store with appropriate expiration times
func (s AuthService) CreateAuth(userID models.UserID, td *models.TokenDetails) (err error) {
	at := time.Unix(td.AtExpires, 0) //converting Unix to UTC(to Time object)
	rt := time.Unix(td.RtExpires, 0)
	now := time.Now()

	err = s.kv.Set(td.AccessUUID, userID.String(), at.Sub(now))
	if err != nil {
		slog.Error("failed to store access token", "error", err, "user_id", userID, "access_uuid", td.AccessUUID)
		return err
	}

	err = s.kv.Set(td.RefreshUUID, userID.String(), rt.Sub(now))
	if err != nil {
		slog.Error("failed to store refresh token", "error", err, "user_id", userID, "refresh_uuid", td.RefreshUUID)
		return err
	}
	return nil
}

// ExtractToken extracts the token from the Authorization header of an HTTP request
func (s AuthService) ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	//normally Authorization the_token_xxx
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}

	return strArr[0]
}

// VerifyToken validates the token signature and returns the parsed token
func (s AuthService) VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := s.ExtractToken(r)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			slog.Error("invalid signing method", "method", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})
	if err != nil {
		slog.Error("failed to verify token", "error", err)
		return nil, err
	}
	return token, nil
}

// TokenValid checks if the token in the request is valid
func (s AuthService) TokenValid(r *http.Request) error {
	token, err := s.VerifyToken(r)
	if err != nil {
		slog.Error("token verification failed", "error", err)
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		slog.Error("invalid token claims")
		return err
	}
	return nil
}

// ExtractTokenMetadata extracts the access details from a verified token
func (s AuthService) ExtractTokenMetadata(r *http.Request) (*models.AccessDetails, error) {
	token, err := s.VerifyToken(r)
	if err != nil {
		slog.Error("failed to verify token for metadata extraction", "error", err)
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			slog.Error("missing access_uuid in token claims")
			return nil, err
		}
		userID := claims["user_id"].(string)

		return &models.AccessDetails{
			AccessUUID: accessUUID,
			UserID:     userID,
		}, nil
	}
	slog.Error("invalid token or claims")
	return nil, err
}

// FetchAuth retrieves the user ID associated with the given access details from the key-value store
func (s AuthService) FetchAuth(authD *models.AccessDetails) (userID models.UserID, err error) {
	rawID, err := s.kv.Get(authD.AccessUUID)
	if err != nil {
		slog.Error("failed to fetch auth details", "error", err, "access_uuid", authD.AccessUUID)
		return userID, err
	}

	userID, err = models.ParseUserID(rawID)
	if err != nil {
		slog.Error("failed to parse user ID", "error", err, "raw_id", rawID)
		return userID, err
	}

	return userID, err
}

// DeleteAuth removes the authentication data for the given UUID from the key-value store
func (s AuthService) DeleteAuth(givenUUID string) (userID models.UserID, err error) {
	rawID, err := s.kv.Del(givenUUID)
	if err != nil {
		slog.Error("failed to delete auth details", "error", err, "uuid", givenUUID)
		return userID, err
	}

	userID, err = models.ParseUserID(rawID)
	if err != nil {
		slog.Error("failed to parse user ID during deletion", "error", err, "raw_id", rawID)
		return userID, err
	}

	return userID, err
}
