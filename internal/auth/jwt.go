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
	c.NotBefore = jwt.NewNumericDate(time.Now())
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c Claims) validate() error {
	if c.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	return nil
}

func (c Claims) String() string {
	val, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling claims: %v", err)
	}
	return string(val)
}

func (c Claims) Token() *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c)
}

func (c Claims) Sign(key HMACBase64Key) (string, error) {
	return c.Token().SignedString([]byte(key))
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

func Subject(sub string) option {
	return func(c *Claims) error {
		c.Subject = sub
		return nil
	}
}

func Audience(aud string) option {
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
