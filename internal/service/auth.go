package service

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var (
	accessSecret  []byte
	refreshSecret []byte
)

// Init loads environment variables once
func Init() error {
	// Try to load .env file, but don't fail if it doesn't exist (e.g., in production)
	if err := godotenv.Load(); err != nil {
		// Log the error but don't necessarily fail
		fmt.Printf("Warning: .env file not found, using environment variables: %v\n", err)
	}

	accessSecret = []byte(os.Getenv("JWT_ACCESS_SECRET"))
	refreshSecret = []byte(os.Getenv("JWT_REFRESH_SECRET"))

	if len(accessSecret) == 0 {
		return errors.New("JWT_ACCESS_SECRET is not set")
	}
	if len(refreshSecret) == 0 {
		return errors.New("JWT_REFRESH_SECRET is not set")
	}

	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashed), nil
}

// CheckPassword compares a password with its hash
func CheckPassword(password, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}

// GenerateAccessToken generates a short-lived access token
func GenerateAccessToken(userID, role string, tokenVersion int32, isBanned bool, isPermanentBan bool) (string, error) {
	if len(accessSecret) == 0 {
		return "", errors.New("access secret not initialized")
	}

	claims := jwt.MapClaims{
		"sub":              userID,
		"role":             role,
		"token_version":    tokenVersion,
		"is_banned":        isBanned,       // ✅ new claim
		"is_permanent_ban": isPermanentBan, // ✅ new claim
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(accessSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a long-lived refresh token
func GenerateRefreshToken(userID string, tokenVersion int32, isBanned bool, isPermanentBan bool) (string, error) {
	if len(refreshSecret) == 0 {
		return "", errors.New("refresh secret not initialized")
	}

	claims := jwt.MapClaims{
		"sub":              userID,
		"token_version":    tokenVersion,
		"is_banned":        isBanned,       // ✅ new claim
		"is_permanent_ban": isPermanentBan, // ✅ new claim
		"exp":              time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":              time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(refreshSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

// VerifyToken parses and validates a JWT token
func VerifyToken(tokenString string, isRefresh bool) (*jwt.Token, error) {
	var secret []byte
	if isRefresh {
		secret = refreshSecret
	} else {
		secret = accessSecret
	}

	if len(secret) == 0 {
		return nil, errors.New("secret not initialized")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}
