package auth

import (
	"slices"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestExpiresA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		t         time.Time
		expectErr bool
	}{
		{
			name:      "future time",
			t:         time.Now().Add(24 * time.Hour),
			expectErr: false,
		},
		{
			name:      "past time",
			t:         time.Now().Add(-24 * time.Hour),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		c := &Claims{}
		err := ExpiresAt(tt.t)(c)
		if tt.expectErr && err == nil {
			t.Errorf("Expected error, got nil")
		}
		if !tt.expectErr && err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}
}

func TestSubject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sub  string
	}{
		{
			name: "valid subject",
			sub:  "test",
		},
	}
	for _, tt := range tests {
		c := &Claims{}
		err := WithSubject(tt.sub)(c)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if c.Subject != tt.sub {
			t.Errorf("Expected subject %q, got %q", tt.sub, c.Subject)
		}
	}
}

func TestAudience(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		aud  string
	}{
		{
			name: "valid audience",
			aud:  "test",
		},
	}
	for _, tt := range tests {
		c := &Claims{}
		err := WithAudience(tt.aud)(c)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if c.Audience[0] != tt.aud {
			t.Errorf("Expected audience %q, got %q", tt.aud, c.Audience[0])
		}
	}
}

func TestHMACBase64Key(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "simple key",
			input: []byte("test"),
		},
		{
			name:  "random key",
			input: MustNewHS256SigningKey(),
		},
	}
	for _, tt := range tests {
		k := HMACBase64Key([]byte(tt.input))
		encoded, err := k.MarshalText()
		if err != nil {
			t.Fatalf("[%s] Unexpected error marshaling: %v", tt.name, err)
		}
		var kd HMACBase64Key
		err = kd.UnmarshalText(encoded)
		if err != nil {
			t.Fatalf("[%s] Unexpected error unmarshaling: %v", tt.name, err)
		}
		if !slices.Equal(k, kd) {
			t.Errorf("[%s] Round-trip mismatch %q, got %q", tt.name, string(k), string(kd))
		}
	}
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()
	defaultKey := MustNewHS256SigningKey()
	baseClaims, _ := NewClaims(
		ExpiresAt(time.Now().Add(24*time.Hour)),
		WithSubject("test"),
		WithAudience("test"),
	)
	var tests = []struct {
		name      string
		claimsF   func() Claims
		key       HMACBase64Key
		verifyKey HMACBase64Key
		expectErr bool
	}{
		{
			name: "valid claims and key",
			claimsF: func() Claims {
				c := *baseClaims
				return c
			},
			key:       defaultKey,
			verifyKey: defaultKey,
			expectErr: false,
		},
		{
			name: "invalid key",
			claimsF: func() Claims {
				c := *baseClaims
				return c
			},
			key:       HMACBase64Key([]byte("invalid")),
			verifyKey: defaultKey,
			expectErr: true,
		},
		{
			name: "expired claims",
			claimsF: func() Claims {
				c := *baseClaims
				c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-24 * time.Hour))
				return c
			},
			key:       defaultKey,
			verifyKey: defaultKey,
			expectErr: true,
		},
		{
			name: "unsupported issuer",
			claimsF: func() Claims {
				c := *baseClaims
				c.Issuer = "somebody else"
				return c
			},
			key:       defaultKey,
			verifyKey: defaultKey,
			expectErr: true,
		},
		{
			name: "no subject",
			claimsF: func() Claims {
				c := *baseClaims
				c.Subject = ""
				return c
			},
			key:       defaultKey,
			verifyKey: defaultKey,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		claims := tt.claimsF()
		token, err := claims.Sign(tt.key)
		if err != nil {
			t.Fatalf("Error signing claims: %v", err)
		}
		claims2, err := VerifyToken(tt.verifyKey, token)
		if (err != nil) != tt.expectErr {
			t.Fatalf("[%s] Unexpected error state verifying claims: %v", tt.name, err)
		}
		if err != nil {
			continue
		}
		if claims.Issuer != claims2.Issuer {
			t.Errorf("Issuer mismatch %q, got %q", claims.Issuer, claims2.Issuer)
		}
		if claims.Subject != claims2.Subject {
			t.Errorf("Subject mismatch %q, got %q", claims.Subject, claims2.Subject)
		}
		if claims.ExpiresAt.Unix() != claims2.ExpiresAt.Unix() {
			t.Errorf("ExpiresAt mismatch %v, got %v", claims.ExpiresAt, claims2.ExpiresAt)
		}
		if !slices.Equal(claims.Audience, claims2.Audience) {
			t.Errorf("Audience mismatch %v, got %v", claims.Audience, claims2.Audience)
		}
		if claims.IssuedAt.Unix() != claims2.IssuedAt.Unix() {
			t.Errorf("IssuedAt mismatch %v, got %v", claims.IssuedAt, claims2.IssuedAt)
		}
	}
}
