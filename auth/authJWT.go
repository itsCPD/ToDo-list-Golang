package authJWT

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// JwtWrapper wraps the signing key and the issuer
type JwtWrapper struct {
	SecretKey      string `default:"verysecret"`
	Issuer         string `default:"Auth"`
	ExpirationMins int64  `default:"2"`
}

// JwtClaim adds email as a claim to the token
type JwtClaim struct {
	Email string
	jwt.StandardClaims
}

// JWT token generation
func (j *JwtWrapper) GenerateToken(email string) (string, error) {
	j = &JwtWrapper{
		SecretKey:      "verysecret",
		Issuer:         "Auth",
		ExpirationMins: 2,
	}
	claims := &JwtClaim{
		Email: email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(j.ExpirationMins)).Unix(),
			Issuer:    j.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(j.SecretKey))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// Validating JWT tokens
func (j *JwtWrapper) ValidateToken(signedToken string) (claims *JwtClaim, err error) {
	j = &JwtWrapper{
		SecretKey: "verysecret",
		Issuer:    "Auth",
	}
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JwtClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(j.SecretKey), nil
		},
	)
	if err != nil {
		return
	}

	claims, ok := token.Claims.(*JwtClaim)
	if !ok {
		err = errors.New("couldn't parse claims")
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("JWT is expired")
		return
	}

	return

}
