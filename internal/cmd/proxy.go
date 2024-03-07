package cmd

import (
	"flag"
	"strings"

	"github.com/efixler/envflags"
)

type Proxy struct {
	proxyURL *envflags.Value[string]
	username *envflags.Value[string]
	password *envflags.Value[string]
}

// Set up the command line args and environment variables for a proxy.
// The assumption here is that the app ultimately supports at least two different proxies,
// one for general usage and another for headless scraping. The general proxy should pass an
// empty proxy name. Other proxies should pass a unique name, which will be used in constructing
// the environment variable and command line argument names.
func AddProxyFlags(proxyName string, flags *flag.FlagSet) *Proxy {
	// TODO: prefix the env var and arg based on whether baseEnv is empty or not.
	var baseEnv, baseArgName string
	helpPrefix := "Default"
	if proxyName != "" {
		baseEnv = strings.ToUpper(proxyName) + "_"
		baseArgName = strings.ToLower(proxyName) + "-"
		helpPrefix = strings.ToUpper(string(proxyName[0])) + strings.ToLower(proxyName[1:])
	}

	proxy := &Proxy{
		proxyURL: envflags.NewString(baseEnv+"PROXY", ""),
		username: envflags.NewString(baseEnv+"PROXY_USERNAME", ""),
		password: envflags.NewString(baseEnv+"PROXY_PASSWORD", ""),
	}
	proxy.proxyURL.AddTo(flags, baseArgName+"proxy", helpPrefix+" proxy URL")
	proxy.username.AddTo(flags, baseArgName+"proxy-username", helpPrefix+" proxy username")
	proxy.password.AddTo(flags, baseArgName+"proxy-password", helpPrefix+" proxy password")
	return proxy
}

func (p *Proxy) ProxyURL() string {
	return p.proxyURL.Get()
}

func (p *Proxy) Username() string {
	return p.username.Get()
}

func (p *Proxy) Password() string {
	return p.password.Get()
}
