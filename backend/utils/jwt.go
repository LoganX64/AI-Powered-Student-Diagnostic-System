package utils

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret string
	jwtExpiryStr string
)

func InitJWTConfig(secret, expiry string) {
	jwtSecret = secret
	jwtExpiryStr = expiry
}

func jwtKey() []byte {
	return []byte(jwtSecret)
}

func jwtExpiry() time.Duration {
	if jwtExpiryStr == "" {
		jwtExpiryStr = "4h"
	}
	expiry, err := time.ParseDuration(jwtExpiryStr)
	if err != nil {
		log.Printf("[JWT] Invalid JWT_EXPIRY format '%s', using default 4h: %v\n", jwtExpiryStr, err)
		return 4 * time.Hour
	}
	return expiry
}

type Claims struct {
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`       // admin | coach | student
	StudentID int    `json:"student_id"` // only for students
	TenantID  int    `json:"tenant_id"`

	jwt.RegisteredClaims
}

func GenerateToken(userID int, role string, studentID int, tenantID int) (string, error) {
	expiry := jwtExpiry()
	claims := Claims{
		UserID:    userID,
		Role:      role,
		StudentID: studentID,
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey())
}

func ValidateToken(tokenStr string) (*Claims, error) {
	log.Println("[JWT] Validating token...")

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		key := jwtKey()
		log.Printf("[JWT] Key used for verification: %d bytes\n", len(key))
		return key, nil
	})

	if err != nil {
		log.Printf("[JWT] Parse error: %v\n", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		log.Printf("[JWT] Claims cast failed\n")
		return nil, jwt.ErrTokenNotValidYet
	}

	log.Printf("[JWT] Token valid: %v, Claims: UserID=%d, Role=%s, TenantID=%d\n", token.Valid, claims.UserID, claims.Role, claims.TenantID)

	if !token.Valid {
		log.Printf("[JWT] Token marked as invalid\n")
		return nil, jwt.ErrTokenNotValidYet
	}

	return claims, nil
}
