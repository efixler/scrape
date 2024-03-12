package cmd

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/efixler/envflags"
)

func TestProxyConfigArgs(t *testing.T) {
	tests := []struct {
		name           string
		envFlagPrefix  string
		addEnabledFlag bool
		proxyName      string
		args           []string
		env            []string
		expected       []string
	}{
		{
			name:           "env with empty proxy name",
			envFlagPrefix:  "",
			addEnabledFlag: false,
			proxyName:      "",
			args:           []string{},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_USERNAME", "user", "PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://proxy.example.com", "user", "pass"},
		},
		{
			name:           "env with proxy name",
			envFlagPrefix:  "",
			addEnabledFlag: false,
			proxyName:      "headless",
			args:           []string{},
			env:            []string{"HEADLESS_PROXY", "http://proxy.example.com", "HEADLESS_PROXY_USERNAME", "user", "HEADLESS_PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://proxy.example.com", "user", "pass"},
		},
		{
			name:           "env with proxy name and env prefix",
			envFlagPrefix:  "SCRAPE_",
			addEnabledFlag: false,
			proxyName:      "headless",
			args:           []string{},
			env:            []string{"SCRAPE_HEADLESS_PROXY", "http://proxy.example.com", "SCRAPE_HEADLESS_PROXY_USERNAME", "user", "SCRAPE_HEADLESS_PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://proxy.example.com", "user", "pass"},
		},
		{
			name:           "args with empty proxy name",
			envFlagPrefix:  "",
			addEnabledFlag: false,
			proxyName:      "",
			args:           []string{"--proxy", "http://argproxy:8080", "--proxy-username", "uarg", "--proxy-password", "parg"},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_USERNAME", "user", "PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://argproxy:8080", "uarg", "parg"},
		},
		{
			name:           "args with proxy name",
			envFlagPrefix:  "",
			addEnabledFlag: false,
			proxyName:      "headless",
			args:           []string{"--headless-proxy", "http://argproxy:8080", "--headless-proxy-username", "uarg", "--headless-proxy-password", "parg"},
			env:            []string{"HEADLESS_PROXY", "http://proxy.example.com", "HEADLESS_PROXY_USERNAME", "user", "HEADLESS_PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://argproxy:8080", "uarg", "parg"},
		},
		{
			name:           "args with proxy name and env prefix",
			envFlagPrefix:  "SCRAPE_",
			addEnabledFlag: false,
			proxyName:      "headless",
			args:           []string{"--headless-proxy", "http://argproxy:8080", "--headless-proxy-username", "uarg", "--headless-proxy-password", "parg"},
			env:            []string{"SCRAPE_HEADLESS_PROXY", "http://proxy.example.com", "SCRAPE_HEADLESS_PROXY_USERNAME", "user", "SCRAPE_HEADLESS_PROXY_PASSWORD", "pass"},
			expected:       []string{"false", "http://argproxy:8080", "uarg", "parg"},
		},
		{
			name:           "env with empty proxy name and enabled flag omitted",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "",
			args:           []string{},
			env:            []string{"PROXY", "http://proxy.example.com"},
			expected:       []string{"false", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with empty proxy name and enabled via ENV",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "",
			args:           []string{},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_ENABLED", "true"},
			expected:       []string{"true", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with empty proxy name and disabled via env",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "",
			args:           []string{},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_ENABLED", "false"},
			expected:       []string{"false", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with empty proxy name and enabled via flag",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "",
			args:           []string{"--use-proxy"},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_ENABLED", "false"},
			expected:       []string{"true", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with empty proxy name and disabled via flag",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "",
			args:           []string{"--use-proxy=0"},
			env:            []string{"PROXY", "http://proxy.example.com", "PROXY_ENABLED", "true"},
			expected:       []string{"false", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with proxy name and enabled via flag",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "headless",
			args:           []string{"--use-headless-proxy"},
			env:            []string{"HEADLESS_PROXY", "http://proxy.example.com", "HEADLESS_PROXY_ENABLED", "false"},
			expected:       []string{"true", "http://proxy.example.com", "", ""},
		},
		{
			name:           "env with proxy name and enabled via env",
			envFlagPrefix:  "",
			addEnabledFlag: true,
			proxyName:      "headless",
			args:           []string{},
			env:            []string{"HEADLESS_PROXY", "http://proxy.example.com", "HEADLESS_PROXY_ENABLED", "true"},
			expected:       []string{"true", "http://proxy.example.com", "", ""},
		},
	}
	for _, tt := range tests {
		envflags.EnvPrefix = tt.envFlagPrefix
		envMap := make(map[string]string)
		for i := 0; i < len(tt.env); i += 2 {
			envMap[tt.env[i]] = tt.env[i+1]
			os.Setenv(tt.env[i], tt.env[i+1])
		}
		flags := flag.NewFlagSet(tt.name, flag.ContinueOnError)
		proxy := AddProxyFlags(tt.proxyName, tt.addEnabledFlag, flags)
		flags.Parse(tt.args)
		if fmt.Sprintf("%v", proxy.Enabled()) != tt.expected[0] {
			t.Errorf("[%s]: Enabled() = %v, want %v", tt.name, proxy.Enabled(), false)
		}
		if proxy.ProxyURL() != tt.expected[1] {
			t.Errorf("[%s]: ProxyURL() = %v, want %v", tt.name, proxy.ProxyURL(), tt.expected[0])
		}
		if proxy.Username() != tt.expected[2] {
			t.Errorf("[%s]: Username() = %v, want %v", tt.name, proxy.Username(), tt.expected[1])
		}
		if proxy.Password() != tt.expected[3] {
			t.Errorf("[%s]: Password() = %v, want %v", tt.name, proxy.Password(), tt.expected[2])
		}

		for ek := range envMap {
			os.Unsetenv(ek)
		}
	}
}
