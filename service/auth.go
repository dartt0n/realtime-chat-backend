package service

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/dartt0n/realtime-chat-backend/models"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type AuthService struct {
	kv kv.KeyValueStore
}

func NewAuthService(kv kv.KeyValueStore) *AuthService {
	return &AuthService{
		kv: kv,
	}
}

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
		return nil, err
	}
	return td, nil
}

// CreateAuth ...
func (s AuthService) CreateAuth(userID models.UserID, td *models.TokenDetails) (err error) {
	at := time.Unix(td.AtExpires, 0) //converting Unix to UTC(to Time object)
	rt := time.Unix(td.RtExpires, 0)
	now := time.Now()

	err = s.kv.Set(td.AccessUUID, userID.String(), at.Sub(now))
	if err != nil {
		return err
	}

	err = s.kv.Set(td.RefreshUUID, userID.String(), rt.Sub(now))
	if err != nil {
		return err
	}
	return nil
}

// ExtractToken ...
func (s AuthService) ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	//normally Authorization the_token_xxx
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

// VerifyToken ...
func (s AuthService) VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := s.ExtractToken(r)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

// TokenValid ...
func (s AuthService) TokenValid(r *http.Request) error {
	token, err := s.VerifyToken(r)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
}

// ExtractTokenMetadata ...
func (s AuthService) ExtractTokenMetadata(r *http.Request) (*models.AccessDetails, error) {
	token, err := s.VerifyToken(r)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, err
		}
		userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return nil, err
		}
		return &models.AccessDetails{
			AccessUUID: accessUUID,
			UserID:     userID,
		}, nil
	}
	return nil, err
}

// FetchAuth ...
func (s AuthService) FetchAuth(authD *models.AccessDetails) (userID models.UserID, err error) {
	rawID, err := s.kv.Get(authD.AccessUUID)
	if err != nil {
		return userID, err
	}

	userID, err = models.ParseUserID(rawID)
	if err != nil {
		return userID, err
	}

	return userID, err
}

// DeleteAuth ...
func (s AuthService) DeleteAuth(authUUID string) (userID models.UserID, err error) {
	rawID, err := s.kv.Del(authUUID)
	if err != nil {
		return userID, err
	}

	userID, err = models.ParseUserID(rawID)
	if err != nil {
		return userID, err
	}

	return userID, err
}
