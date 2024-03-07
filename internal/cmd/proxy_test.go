package cmd

import (
	"flag"
	"os"
	"testing"
)

func TestProxyArgs(t *testing.T) {
	// TODO: Figure out how to test FlagSets
	// TODO: Make this test better
	tests := []struct {
		name      string
		proxyName string
		env       []string
	}{
		{
			name:      "test with empty proxy name",
			proxyName: "",
			env:       []string{"PROXY", "http://proxy.example.com", "PROXY_USERNAME", "user", "PROXY_PASSWORD", "pass"},
		},
		{
			name:      "test with proxy name",
			proxyName: "headless",
			env:       []string{"HEADLESS_PROXY", "http://proxy.example.com", "HEADLESS_PROXY_USERNAME", "user", "HEADLESS_PROXY_PASSWORD", "pass"},
		},
	}
	for _, tt := range tests {
		envMap := make(map[string]string)
		for i := 0; i < len(tt.env); i += 2 {
			envMap[tt.env[i]] = tt.env[i+1]
			os.Setenv(tt.env[i], tt.env[i+1])
		}
		flags := flag.NewFlagSet(tt.name, flag.ContinueOnError)
		proxy := AddProxyFlags(tt.proxyName, flags)
		if proxy.ProxyURL() != envMap[tt.env[0]] {
			t.Errorf("[%s]: ProxyURL() = %v, want %v", tt.name, proxy.ProxyURL(), envMap[tt.env[0]])
		}
		if proxy.Username() != envMap[tt.env[2]] {
			t.Errorf("[%s]: Username() = %v, want %v", tt.name, proxy.Username(), envMap[tt.env[2]])
		}
		if proxy.Password() != envMap[tt.env[4]] {
			t.Errorf("[%s]: Password() = %v, want %v", tt.name, proxy.Password(), envMap[tt.env[4]])
		}
		for ek := range envMap {
			os.Unsetenv(ek)
		}
	}
}
