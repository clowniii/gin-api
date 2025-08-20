package jwt

import (
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret []byte
	expire time.Duration
	issuer string
}

type Claims struct {
	UserID int64   `json:"sub"`
	Roles  []int64 `json:"roles"`
	JTI    string  `json:"jti"`
	jwtlib.RegisteredClaims
}

func NewManager(secret string, expireSeconds int, issuer string) *Manager {
	return &Manager{secret: []byte(secret), expire: time.Duration(expireSeconds) * time.Second, issuer: issuer}
}

func (m *Manager) Generate(userID int64, roles []int64, jti string) (string, error) {
	claims := Claims{
		UserID: userID,
		Roles:  roles,
		JTI:    jti,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(m.expire)),
		},
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtlib.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwtlib.ErrTokenInvalidClaims
}

func (m *Manager) ExpireDuration() time.Duration { return m.expire }
