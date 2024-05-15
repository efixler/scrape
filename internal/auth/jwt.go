// JWT token generation and verification logic.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "scrape"
)

type option func(*Claims) error

type Claims struct {
	jwt.RegisteredClaims
}

func NewClaims(options ...option) (*Claims, error) {
	c := &Claims{}
	c.Issuer = Issuer
	c.IssuedAt = jwt.NewNumericDate(time.Now())
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Implements the jwt.ClaimsValidator interface
// In addition to the explicit call when the claims are created,
// this method is called by ParseWithClaims when the token is parsed.
func (c Claims) Validate() error {
	if c.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	return nil
}

func (c Claims) String() string {
	val, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Sprintf("error stringifying claims: %v", err)
	}
	return string(val)
}

func (c Claims) Token() *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c)
}

func (c Claims) Sign(key HMACBase64Key) (string, error) {
	return c.Token().SignedString([]byte(key))
}

func ExpiresIn(d time.Duration) option {
	return func(c *Claims) error {
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(d))
		return nil
	}
}

func ExpiresAt(t time.Time) option {
	return func(c *Claims) error {
		if t.Before(time.Now()) {
			return fmt.Errorf("expiration time %v is in the past", t)
		}
		c.ExpiresAt = jwt.NewNumericDate(t)
		return nil
	}
}

func WithSubject(sub string) option {
	return func(c *Claims) error {
		c.Subject = sub
		return nil
	}
}

func WithAudience(aud string) option {
	return func(c *Claims) error {
		c.Audience = []string{aud}
		return nil
	}
}

func MustNewHS256SigningKey() HMACBase64Key {
	key, err := NewHS256SigningKey()
	if err != nil {
		panic(err)
	}
	return key
}

func NewHS256SigningKey() (HMACBase64Key, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	return HMACBase64Key(key), err
}

type HMACBase64Key []byte

func (b HMACBase64Key) MarshalText() ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(b))
	return []byte(encoded), nil
}

func (b *HMACBase64Key) UnmarshalText(text []byte) error {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(text)))
	n, err := base64.StdEncoding.Decode(decoded, text)
	if err != nil {
		return err
	}
	*b = HMACBase64Key(decoded[:n])
	return nil
}

var parser *jwt.Parser

// VerifyToken verifies the token string using the provided key.
// In the case where the token's signature is invalid, the function will not return any claims.
func VerifyToken(key HMACBase64Key, tokenString string) (*Claims, error) {
	if parser == nil {
		parser = makeParser()
	}
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type: %T", token.Claims)
	}
	return claims, nil
}

func makeParser() *jwt.Parser {
	parser := jwt.NewParser(
		jwt.WithIssuer(Issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithLeeway(1*time.Minute),
		jwt.WithIssuedAt(),
	)
	return parser
}
