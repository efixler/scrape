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
	if proxyName != "" {
		baseEnv = strings.ToUpper(proxyName) + "_"
		baseArgName = strings.ToLower(proxyName) + "-"
	}

	proxy := &Proxy{
		proxyURL: envflags.NewString(baseEnv+"PROXY", ""),
		username: envflags.NewString(baseEnv+"PROXY_USERNAME", ""),
		password: envflags.NewString(baseEnv+"PROXY_PASSWORD", ""),
	}
	flags.Var(proxy.proxyURL, baseArgName+"proxy", "Proxy URL")
	flags.Var(proxy.username, baseArgName+"proxy-username", "Proxy username")
	flags.Var(proxy.password, baseArgName+"proxy-password", "Proxy password")
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
