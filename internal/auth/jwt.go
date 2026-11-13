package auth

import (
	"errors"
	"time"

	"gin-demo/config"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenUser    TokenType = "user"
	TokenVehicle TokenType = "vehicle"
)

type Claims struct {
	Sub  string    `json:"sub"` // 用户ID 或 车辆ID
	Role string    `json:"role"`
	Typ  TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// GenerateUserToken 签发用户令牌（含角色，但角色不允许请求体覆盖）
func GenerateUserToken(uid uint, role string) (string, error) {
	c := config.Get()
	claims := Claims{
		Sub:  itoa(uid),
		Role: role,
		Typ:  TokenUser,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(c.JWTSecret))
}

// GenerateVehicleToken 签发车端令牌（与用户令牌完全隔离，使用不同密钥）
func GenerateVehicleToken(vid string) (string, error) {
	c := config.Get()
	claims := Claims{
		Sub: vid,
		Typ: TokenVehicle,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(c.VehicleSecret))
}

func parse(tokenStr string, secret string) (*Claims, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func ParseUserToken(tokenStr string) (*Claims, error) {
	c := config.Get()
	cl, err := parse(tokenStr, c.JWTSecret)
	if err != nil {
		return nil, err
	}
	if cl.Typ != TokenUser {
		return nil, errors.New("not a user token")
	}
	return cl, nil
}

func ParseVehicleToken(tokenStr string) (*Claims, error) {
	c := config.Get()
	cl, err := parse(tokenStr, c.VehicleSecret)
	if err != nil {
		return nil, err
	}
	if cl.Typ != TokenVehicle {
		return nil, errors.New("not a vehicle token")
	}
	return cl, nil
}

func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var b []byte
	for u > 0 {
		b = append([]byte{byte('0' + u%10)}, b...)
		u /= 10
	}
	return string(b)
}
