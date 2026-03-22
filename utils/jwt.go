package utils

import (
	"fmt"
	"sync"
	"time"

	"transit-server/config"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims extends JWT standard claims with user-specific fields.
type CustomClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// tokenBlacklist stores revoked tokens (in-memory).
var tokenBlacklist sync.Map

// GenerateAccessToken creates a short-lived JWT access token.
func GenerateAccessToken(userID uint, email, role string) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWTAccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "transit-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// GenerateRefreshToken creates a long-lived JWT refresh token.
func GenerateRefreshToken(userID uint, email, role string) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWTRefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "transit-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// ValidateToken parses and validates a JWT token string.
func ValidateToken(tokenString string) (*CustomClaims, error) {
	// Check if token has been blacklisted (logged out)
	if _, blacklisted := tokenBlacklist.Load(tokenString); blacklisted {
		return nil, fmt.Errorf("token has been revoked")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// BlacklistToken adds a token to the blacklist (used for logout).
func BlacklistToken(tokenString string) {
	tokenBlacklist.Store(tokenString, true)
}
