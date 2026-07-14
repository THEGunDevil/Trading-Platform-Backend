package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// Keep secrets in environment variables in production
var JwtAccessSecret = []byte("super-secret-access-key")
var JwtRefreshSecret = []byte("super-secret-refresh-key")

// Hash and verify passwords
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func CheckPassword(password, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}

// -------------------------------
// JWT Functions
// -------------------------------

// GenerateAccessToken: short-lived (~15 minutes)
func GenerateAccessToken(userID, role string, tokenVersion int32) (string, error) {
	claims := jwt.MapClaims{
		"sub":           userID,
		"role":          role,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtAccessSecret)
}

// GenerateRefreshToken: long-lived (~7 days)
func GenerateRefreshToken(userID string, tokenVersion int32) (string, error) {
	claims := jwt.MapClaims{
		"sub":           userID,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtRefreshSecret)
}

// Parse and validate JWT
func VerifyToken(tokenString string, isRefresh bool) (*jwt.Token, error) {
	var secret []byte
	if isRefresh {
		secret = JwtRefreshSecret
	} else {
		secret = JwtAccessSecret
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure token method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}