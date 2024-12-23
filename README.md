# cli

[![GoDoc](https://godoc.org/github.com/mfridman/cli?status.svg)](https://godoc.org/github.com/mfridman/cli)
[![Go Report
Card](https://goreportcard.com/badge/github.com/mfridman/cli)](https://goreportcard.com/report/github.com/mfridman/cli)

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

Here's a simple example of a todo list manager CLI:

```go
root := &cli.Command{
	Name:        "echo",
	Usage:       "echo <text...> [flags]",
	Description: "Echo is a simple command that echoes back the input",
	Flags: cli.FlagSetFunc(func(f *flag.FlagSet) {
		f.Bool("c", false, "capitalize the input")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		if len(s.Args) == 0 {
			return &cli.HelpError{Err: errors.New("missing input to echo")}
		}
		output := strings.Join(s.Args, " ")
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
	Name        string
	Exec        func(ctx context.Context, s *State) error
	Usage       string
	Description string
	Flags       *flag.FlagSet
	SubCommands []*Command
	UsageFunc   func(*Command) string
}
```

The `Name` field is the command's name and is **required**.

The `Usage` and `Description` fields are used to generate help text. Nice-to-have but not required.

The `Flags` field is a `*flag.FlagSet` that defines the command's flags. The `SubCommands` field is
a slice of child commands.

> [!TIP]
>
> There's a top-level convenience function `FlagSetFunc` that allows you to define flags inline:

```go
cmd.Flags = cli.FlagSetFunc(func(fs *flag.FlagSet) {
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
	Flags: cli.FlagSetFunc(func(f *flag.FlagSet) {
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

- `<required>` Required argument
- `[optional]` Optional argument
- `<arg...>` One or more arguments
- `[arg...]` Zero or more arguments
- `(a|b)` Must choose one of a or b
- `[-f <file>]` Flag with value (optional)
- `-f <file>` Flag with value (required)

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

## Future Work

- [ ] Add required flags support
- [ ] Improve Help text generation (consider a `clihelp` package for standalone use)

## Status

This project is in active development. Please open an issue if you encounter any problems or have
suggestions for improvement.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
