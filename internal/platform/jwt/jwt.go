package jwt

import (
	"errors"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Issuer signs and verifies HS256 access tokens.
type Issuer struct {
	secret []byte
	expiry time.Duration
}

func NewIssuer(secret string, expiry time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), expiry: expiry}
}

func (i *Issuer) Issue(userID string) (string, int64, error) {
	now := time.Now()
	claims := gojwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  gojwt.NewNumericDate(now),
		ExpiresAt: gojwt.NewNumericDate(now.Add(i.expiry)),
	}

	signed, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", 0, err
	}
	return signed, int64(i.expiry.Seconds()), nil
}

// Parse validates a token's signature and expiry and returns its subject.
func (i *Issuer) Parse(tokenString string) (string, error) {
	claims := &gojwt.RegisteredClaims{}
	_, err := gojwt.ParseWithClaims(tokenString, claims, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", errors.New("token has no subject")
	}
	return claims.Subject, nil
}
