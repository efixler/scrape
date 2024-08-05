// Verify and decode auth keys for the scrape service.
//
// Run `scrape-jwt-decode -h` for complete help and command line options.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/internal/auth"
)

var (
	flags      flag.FlagSet
	signingKey *envflags.Value[*auth.HMACBase64Key]
)

func main() {
	token := flags.Arg(0)
	key := *signingKey.Get()
	if len(key) == 0 {
		slog.Error("No signing key provided")
		os.Exit(1)
	}
	claims, err := auth.VerifyToken(key, token)
	if err != nil {
		slog.Error("Error verifying token, the token or signature are invalid", "err", err)
		os.Exit(1)
	}
	fmt.Println("\nThis JWT is valid. Claims:\n------")
	fmt.Println(claims)
}

func init() {
	flags.Init("scrape-jwt-encode", flag.ExitOnError)
	flags.Usage = usage
	envflags.EnvPrefix = "SCRAPE_"
	signingKey = envflags.NewText("SIGNING_KEY", &auth.HMACBase64Key{})
	signingKey.AddTo(&flags, "signing-key", "HS256 key to sign the JWT token")
	flags.Parse(os.Args[1:])
	if len(flags.Args()) == 0 {
		slog.Error("Token is required")
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`
Verify and decode scrape JWT tokens.

Signing key is required to verify the token signature. The signing key should be base64
encoded and can be provided as a command line flag or environment variable.

Usage: 
-----
scrape-jwt-decode [-signing-key keyval] token
	`)

	flags.PrintDefaults()
}
