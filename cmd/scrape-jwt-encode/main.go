// scrape-jwt-encode generates auth keys for the scrape service
//
// Run `scrape-jwt-encode -h` for complete help and command line options.

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/internal/auth"
)

var (
	flags      flag.FlagSet
	expires    = time.Now().Add(24 * time.Hour * 365)
	subject    string
	audience   = "moz"
	signingKey *envflags.Value[*auth.HMACBase64Key]
	makeKey    bool
)

func main() {
	if makeKey {
		makeSigningKey()
		return
	}
	makeToken()
}

func makeToken() {
	if subject == "" {
		slog.Error("Subject is required")
		usage()
		os.Exit(1)
	}
	claims, err := auth.NewClaims(
		auth.ExpiresAt(expires),
		auth.WithSubject(subject),
		auth.WithAudience(audience),
	)
	if err != nil {
		slog.Error("Error generating claims", "err", err)
		os.Exit(1)
	}
	fmt.Println("\nClaims:\n------")
	fmt.Println(claims)
	key := *signingKey.Get()
	if len(key) == 0 {
		slog.Warn("No signing key provided, cannot sign token, exiting")
		os.Exit(1)
	}
	ss, err := claims.Sign(key)
	if err != nil {
		slog.Error("Error signing token", "err", err)
		os.Exit(1)
	}
	fmt.Println("\nToken:\n-----")
	fmt.Println(ss)
}

func makeSigningKey() {
	key, err := auth.NewHS256SigningKey()
	if err != nil {
		slog.Error("Error generating signing key", "err", err)
		os.Exit(1)
	}
	encoded, err := key.MarshalText()
	if err != nil {
		slog.Error("Error encoding signing key", "err", err)
		os.Exit(1)
	}
	fmt.Println("Be sure to save this key, as it can't be re-generated:")
	fmt.Println(string(encoded))
}

func init() {
	flags.Init("scrape-jwt-encode", flag.ExitOnError)
	flags.Usage = usage
	envflags.EnvPrefix = "SCRAPE_"
	flags.BoolVar(&makeKey, "make-key", false, "Generate a new signing key")
	flags.TextVar(&expires, "exp", expires, "Expiration date for the key, in RFC3339 format. Default is 1 year from now.")
	flags.StringVar(&subject, "sub", "", "Subject (holder name) for the key (required)")
	flags.StringVar(&audience, "aud", audience, "Audience (recipient) for the key")
	signingKey = envflags.NewText("SIGNING_KEY", &auth.HMACBase64Key{})
	signingKey.AddTo(&flags, "signing-key", "HS256 key to sign the JWT token")
	flags.Parse(os.Args[1:])
}

func usage() {
	fmt.Println(`
Generates JWT tokens for the scrape service. Also makes the signing key to use for the tokens.

Usage: 
-----
scrape-jwt-encode -sub subject [-signing-key key] [-exp expiration] [-aud audience]
scrape-jwt-encode -make-key
	`)

	flags.PrintDefaults()
}
