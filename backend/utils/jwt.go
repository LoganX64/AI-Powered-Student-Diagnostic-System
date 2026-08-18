package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
	expiry time.Duration
	issuer string
}

func NewJWTManager(secret, expiryStr, issuer string) *JWTManager {
	if expiryStr == "" {
		expiryStr = "4h"
	}
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		log.Printf("[JWT] Invalid JWT_EXPIRY format '%s', using default 4h: %v\n", expiryStr, err)
		expiry = 4 * time.Hour
	}
	return &JWTManager{
		secret: []byte(secret),
		expiry: expiry,
		issuer: issuer,
	}
}

var defaultManager *JWTManager

func InitJWTConfig(secret, expiry, issuer string) {
	defaultManager = NewJWTManager(secret, expiry, issuer)
}

func jwtManager() *JWTManager {
	return defaultManager
}

type Claims struct {
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`       // admin | coach | student
	StudentID int    `json:"student_id"` // only for students
	TenantID  int    `json:"tenant_id"`

	jwt.RegisteredClaims
}

func (m *JWTManager) GenerateToken(userID int, role string, studentID int, tenantID int) (string, error) {
	claims := Claims{
		UserID:    userID,
		Role:      role,
		StudentID: studentID,
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{m.issuer},
			Issuer:    m.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return m.secret, nil
	})

	if err != nil {
		log.Printf("[JWT] Parse error: %v\n", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrTokenNotValidYet
	}

	if !token.Valid {
		log.Printf("[JWT] Token marked as invalid for UserID=%d\n", claims.UserID)
		return nil, jwt.ErrTokenNotValidYet
	}

	if claims.Issuer != m.issuer {
		log.Printf("[JWT] Invalid issuer: %s\n", claims.Issuer)
		return nil, fmt.Errorf("invalid issuer")
	}
	audValid := false
	for _, a := range claims.Audience {
		if a == m.issuer {
			audValid = true
			break
		}
	}
	if !audValid {
		log.Printf("[JWT] Invalid audience: %v\n", claims.Audience)
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}

func GenerateToken(userID int, role string, studentID int, tenantID int) (string, error) {
	return jwtManager().GenerateToken(userID, role, studentID, tenantID)
}

func ValidateToken(tokenStr string) (*Claims, error) {
	return jwtManager().ValidateToken(tokenStr)
}
