package auth

import (
	"slices"
	"testing"
	"time"
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
		err := Subject(tt.sub)(c)
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
		err := Audience(tt.aud)(c)
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
		k := HMACBase64Key(tt.input)
		encoded, err := k.MarshalText()
		if err != nil {
			t.Errorf("[%s] Unexpected error marshaling: %v", tt.name, err)
		}
		var kd HMACBase64Key
		err = kd.UnmarshalText(encoded)
		if err != nil {
			t.Errorf("[%s] Unexpected error unmarshaling: %v", tt.name, err)
		}
		if !slices.Equal(k, kd) {
			t.Errorf("[%s] Round-trip mismatch %q, got %q", tt.name, string(k), string(kd))
		}
		t.Logf("Encoded: %q", string(encoded))
	}
}
