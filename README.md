# cli

[![GoDoc](https://godoc.org/github.com/mfridman/cli?status.svg)](https://godoc.org/github.com/mfridman/cli)
[![CI](https://github.com/mfridman/cli/actions/workflows/ci.yaml/badge.svg)](https://github.com/mfridman/cli/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mfridman/cli)](https://goreportcard.com/report/github.com/mfridman/cli)

A lightweight framework for building Go CLI applications with nested subcommands.

Supports flexible flag placement ([allowing flags anywhere on the
CLI](https://mfridman.com/blog/2024/allowing-flags-anywhere-on-the-cli/)), since Go's standard
library requires flags before arguments.

## Features

- **Nested Commands**: Build hierarchical command structures (like `git remote add`)
- **Flexible Flag Parsing**: Supports flags and arguments in any order
  - **Inherited Flags**: Child commands can access parent flags
  - **Type-Safe Flags**: Type-inferred flag accessors
- **Built-in Help**: Automatic help text generation
- **Auto Suggestions**: Suggests similar commands when users make typos

## Installation

```bash
go get github.com/mfridman/cli@latest
```

Required go version: 1.21 or higher

## Quick Start

Here's a simple example of a CLI application that echoes back the input:

```go
root := &cli.Command{
	Name:      "echo",
	Usage:     "echo <text...> [flags]",
	ShortHelp: "echo is a simple command that prints the provided text",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		// Add a flag to capitalize the input
		f.Bool("c", false, "capitalize the input")
	}),
	FlagsMetadata: []cli.FlagMetadata{
		{Name: "c", Required: true},
	},
	Exec: func(ctx context.Context, s *cli.State) error {
		if len(s.Args) == 0 {
			// Return a new error with the error code ErrShowHelp
			return fmt.Errorf("no text provided")
		}
		output := strings.Join(s.Args, " ")
		// If -c flag is set, capitalize the output
		if cli.GetFlag[bool](s, "c") {
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
```

This code defines a simple `echo` command that echoes back the input. It supports a `-c` flag to
capitalize the input.

## Command Structure

Each command in your CLI application is represented by a `Command` struct:

```go
type Command struct {
	Name          string
	Usage         string
	ShortHelp     string
	UsageFunc     func(*Command) string
	Flags         *flag.FlagSet
	FlagsMetadata []FlagMetadata
	SubCommands   []*Command
	Exec          func(ctx context.Context, s *State) error
}
```

The `Name` field is the command's name and is **required**.

The `Usage` and `ShortHelp` fields are used to generate help text. Nice-to-have but not required.

The `Flags` field is a `*flag.FlagSet` that defines the command's flags. The `SubCommands` field is
a slice of child commands.

> [!TIP]
>
> There's a top-level convenience function `FlagsFunc` that allows you to define flags inline:

```go
cmd.Flags = cli.FlagsFunc(func(fs *flag.FlagSet) {
	fs.Bool("verbose", false, "enable verbose output")
	fs.String("output", "", "output file")
	fs.Int("count", 0, "number of items")
})
```

The `Exec` field is a function that is called when the command is executed. This is where you put
your business logic.

## Flag Access

Flags can be accessed using the type-safe `GetFlag` function, called inside your `Exec` function:

```go
// Access boolean flag
verbose := cli.GetFlag[bool](state, "verbose")

// Access string flag
output := cli.GetFlag[string](state, "output")

// Access integer flag
count := cli.GetFlag[int](state, "count")
```

### State Inheritance

Child commands automatically inherit their parent command's flags:

```go
// Parent command with a verbose flag
root := cli.Command{
	Name: "root",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		f.Bool("verbose", false, "enable verbose mode")
	}),
}

// Child command that can access parent's verbose flag
sub := cli.Command{
	Name: "sub",
	Exec: func(ctx context.Context, s *cli.State) error {
		verbose := cli.GetFlag[bool](s, "verbose")
		if verbose {
			fmt.Println("Verbose mode enabled")
		}
		return nil
	},
}
```

## Help System

Help text is automatically generated, but you can customize it by setting the `UsageFunc` field.

## Usage Syntax Conventions

When reading command usage strings, the following syntax is used:

| Syntax        | Description                |
| ------------- | -------------------------- |
| `<required>`  | Required argument          |
| `[optional]`  | Optional argument          |
| `<arg...>`    | One or more arguments      |
| `[arg...]`    | Zero or more arguments     |
| `(a\|b)`      | Must choose one of a or b  |
| `[-f <file>]` | Flag with value (optional) |
| `-f <file>`   | Flag with value (required) |

Examples:

```bash
# Two required arguments
copy <source> <dest>
# Zero or more paths
ls [path...]
# Optional flag with value, required host
ssh [-p <port>] <user@host>
# Required subcommand, optional remote
git (pull|push) [remote]
```

## Status

This project is in active development and undergoing changes as the API is refined. Please open an
issue if you encounter any problems or have suggestions for improvement.

- [x] Nail down required flags implementation
- [ ] Add tests for typos and command suggestions, crude levenstein distance for now
- [ ] Internal implementation (not user-facing), track selected `*Command` in `*State` and remove
      `flags  *flag.FlagSet` from `*State`
- [ ] Figure out whether to keep `*Error` and whether to catch `ErrShowHelp` in `ParseAndRun`
- [ ] Should `Parse`, `Run` and `ParseAndRun` be methods on `*Command`?
- [ ] What to do with `showHelp()`, should it be a standalone function or an exported method on
      `*Command`?
- [ ] Is there room for `clihelp` package for standalone use?

## Acknowledgements

There are many great CLI libraries out there, but I always felt [they were too heavy for my
needs](https://mfridman.com/blog/2021/a-simpler-building-block-for-go-clis/).

I was inspired by Peter Bourgon's [ff](https://github.com/peterbourgon/ff) library, specifically the
`v3` branch, which was soooo close to what I wanted. But the `v4` branch took a different direction
and I wanted to keep the simplicity of the `v3` branch. This library aims to pick up where `ff/v3`
left off.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
