package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mfridman/cli"
)

func main() {
	root := &cli.Command{
		Name:      "echo",
		Usage:     "echo <text...> [flags]",
		ShortHelp: "echo is a simple command that prints the provided text",
		Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
			// Add a flag to capitalize the input
			f.Bool("c", false, "capitalize the input")
		}),
		RequiredFlags: []string{
			"c",
		},
		Exec: func(ctx context.Context, s *cli.State) error {
			if len(s.Args) == 0 {
				// Return a new error with the error code ErrShowHelp
				return fmt.Errorf("no text provided")
			}
			output := strings.Join(s.Args, " ")
			// If -c flag is set, capitalize the output
			if cli.GetFlag[bool](s, "c") || cli.GetFlag[bool](s, "capitalize") {
				output = strings.ToUpper(output)
			}
			fmt.Fprintln(s.Stdout, output)
			return nil
		},
	}
	err := cli.ParseAndRun(context.Background(), root, os.Args[1:], nil)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
