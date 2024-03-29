package resource

import "testing"

func TestFetchMethodString(t *testing.T) {
	tests := []struct {
		name string
		f    FetchMethod
		want string
	}{
		{
			name: "Client",
			f:    Client,
			want: "client",
		},
		{
			name: "Headless",
			f:    Headless,
			want: "headless",
		},
		{
			name: "Unknown",
			f:    3,
			want: "unknown",
		},
	}
	for _, tt := range tests {
		if got := tt.f.String(); got != tt.want {
			t.Errorf("[%s] FetchMethod.String() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
