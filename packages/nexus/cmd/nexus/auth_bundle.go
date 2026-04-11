package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/inizio/nexus/packages/nexus/pkg/runtime/authbundle"
)

func runAuthBundleCommand(args []string) {
	fs := flag.NewFlagSet("auth-bundle", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outPath := fs.String("output", "", "write base64-encoded bundle to this file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: nexus auth-bundle [--output path]")
		os.Exit(2)
	}

	b64, err := authbundle.BuildFromHome()
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth-bundle: %v\n", err)
		os.Exit(1)
	}
	if strings.TrimSpace(b64) == "" {
		fmt.Fprintln(os.Stderr, "auth-bundle: empty bundle (no registry-matched files under $HOME)")
		os.Exit(0)
	}
	if *outPath != "" {
		if err := os.WriteFile(*outPath, []byte(b64), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "auth-bundle: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "auth-bundle: wrote %d base64 characters to %s\n", len(b64), *outPath)
		return
	}
	fmt.Print(b64)
}
