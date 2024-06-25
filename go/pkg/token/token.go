package token

import "github.com/golang-jwt/jwt/v5"

func Token(sub string, exp *jwt.NumericDate) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "k8s",
		Audience:  jwt.ClaimStrings{"controller"},
		Subject:   sub,
		ExpiresAt: exp,
	})

	signed, err := token.SignedString([]byte("k8s-key"))
	if err != nil {
		panic(err)
	}

	return signed
}
