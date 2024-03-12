package cmd

import (
	"flag"
	"strings"

	"github.com/efixler/envflags"
)

type ProxyFlags struct {
	enabled  *envflags.Value[bool]
	proxyURL *envflags.Value[string]
	username *envflags.Value[string]
	password *envflags.Value[string]
}

func (p *ProxyFlags) addEnabledFlag(proxyName string, flags *flag.FlagSet) {
	// --use-xxx-proxy
	var argName, envName string
	if proxyName == "" {
		argName = "use-proxy"
		envName = "PROXY_ENABLED"
		proxyName = "default"
	} else {
		argName = "use-" + strings.ToLower(proxyName) + "-proxy"
		envName = strings.ToUpper(proxyName) + "_PROXY_ENABLED"
	}

	p.enabled = envflags.NewBool(envName, false)
	p.enabled.AddTo(flags, argName, "Use the "+proxyName+" proxy")
}

// Set up the command line args and environment variables for a proxy.
// The assumption here is that the app ultimately supports at least two different proxies,
// one for general usage and another for headless scraping. The general proxy should pass an
// empty proxy name. Other proxies should pass a unique name, which will be used in constructing
// the environment variable and command line argument names.
func AddProxyFlags(proxyName string, withEnabledFlag bool, flags *flag.FlagSet) *ProxyFlags {
	var baseEnv, baseArgName string
	helpPrefix := "Default"
	if proxyName != "" {
		baseEnv = strings.ToUpper(proxyName) + "_"
		baseArgName = strings.ToLower(proxyName) + "-"
		helpPrefix = strings.ToUpper(string(proxyName[0])) + strings.ToLower(proxyName[1:])
	}

	proxy := &ProxyFlags{
		proxyURL: envflags.NewString(baseEnv+"PROXY", ""),
		username: envflags.NewString(baseEnv+"PROXY_USERNAME", ""),
		password: envflags.NewString(baseEnv+"PROXY_PASSWORD", ""),
	}
	proxy.proxyURL.AddTo(flags, baseArgName+"proxy", helpPrefix+" proxy URL")
	proxy.username.AddTo(flags, baseArgName+"proxy-username", helpPrefix+" proxy username")
	proxy.password.AddTo(flags, baseArgName+"proxy-password", helpPrefix+" proxy password")

	if withEnabledFlag {
		proxy.addEnabledFlag(proxyName, flags)
	}

	return proxy
}

func (p *ProxyFlags) Enabled() bool {
	if p.enabled == nil {
		return false
	}
	return p.enabled.Get()
}

func (p *ProxyFlags) ProxyURL() string {
	return p.proxyURL.Get()
}

func (p *ProxyFlags) Username() string {
	return p.username.Get()
}

func (p *ProxyFlags) Password() string {
	return p.password.Get()
}
