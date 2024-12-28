package cli

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/mfridman/cli/pkg/suggest"
	"github.com/mfridman/cli/pkg/textutil"
)

// NoExecError is returned when a command has no execution function.
type NoExecError struct {
	Command *Command
}

func (e *NoExecError) Error() string {
	return fmt.Sprintf("command %q has no execution function", getCommandPath(e.Command.state.commandPath))
}

// Command represents a CLI command or subcommand within the application's command hierarchy.
type Command struct {
	// Name is always a single word representing the command's name. It is used to identify the
	// command in the command hierarchy and in help text.
	Name string

	// Usage provides the command's full usage pattern.
	//
	// Example: "cli todo list [flags]"
	Usage string

	// ShortHelp is a brief description of the command's purpose. It is displayed in the help text
	// when the command is shown.
	ShortHelp string

	// UsageFunc is an optional function that can be used to generate a custom usage string for the
	// command. It receives the current command and should return a string with the full usage
	// pattern.
	UsageFunc func(*Command) string

	// Flags holds the command-specific flag definitions. Each command maintains its own flag set
	// for parsing arguments.
	Flags *flag.FlagSet
	// FlagsMetadata is an optional list of flag information to extend the FlagSet with additional
	// metadata. This is useful for tracking required flags.
	FlagsMetadata []FlagMetadata

	// SubCommands is a list of nested commands that exist under this command.
	SubCommands []*Command

	// Exec defines the command's execution logic. It receives the current application [State] and
	// returns an error if execution fails. This function is called when [Run] is invoked on the
	// command.
	Exec func(ctx context.Context, s *State) error

	state *State
}

func (c *Command) terminal() (*Command, *State) {
	if c.state == nil || len(c.state.commandPath) == 0 {
		return c, c.state
	}

	// Get the last command in the path - this is our terminal command
	terminalCmd := c.state.commandPath[len(c.state.commandPath)-1]
	return terminalCmd, c.state
}

// FlagMetadata holds additional metadata for a flag, such as whether it is required.
type FlagMetadata struct {
	// Name is the flag's name. Must match the flag name in the flag set.
	Name string

	// Required indicates whether the flag is required.
	Required bool
}

// FlagsFunc is a helper function that creates a new [flag.FlagSet] and applies the given function
// to it. Intended for use in command definitions to simplify flag setup. Example usage:
//
//	cmd.Flags = cli.FlagsFunc(func(f *flag.FlagSet) {
//	    f.Bool("verbose", false, "enable verbose output")
//	    f.String("output", "", "output file")
//	    f.Int("count", 0, "number of items")
//	})
func FlagsFunc(fn func(*flag.FlagSet)) *flag.FlagSet {
	fset := flag.NewFlagSet("", flag.ContinueOnError)
	fn(fset)
	return fset
}

// findSubCommand searches for a subcommand by name and returns it if found. Returns nil if no
// subcommand with the given name exists.
func (c *Command) findSubCommand(name string) *Command {
	for _, sub := range c.SubCommands {
		if strings.EqualFold(sub.Name, name) {
			return sub
		}
	}
	return nil
}

func (c *Command) formatUnknownCommandError(unknownCmd string) error {
	var known []string
	for _, sub := range c.SubCommands {
		known = append(known, sub.Name)
	}
	suggestions := suggest.FindSimilar(unknownCmd, known, 3)
	if len(suggestions) > 0 {
		return fmt.Errorf("unknown command %q. Did you mean one of these?\n\t%s",
			unknownCmd,
			strings.Join(suggestions, "\n\t"))
	}
	return fmt.Errorf("unknown command %q", unknownCmd)
}

func defaultUsage(c *Command) string {
	var b strings.Builder

	// Handle custom usage function
	if c.UsageFunc != nil {
		return c.UsageFunc(c)
	}

	// Short help section
	if c.ShortHelp != "" {
		for _, line := range textutil.Wrap(c.ShortHelp, 80) {
			b.WriteString(line)
			b.WriteRune('\n')
		}
		b.WriteRune('\n')
	}

	// Usage section
	b.WriteString("Usage:\n  ")
	if c.Usage != "" {
		b.WriteString(c.Usage)
		b.WriteRune('\n')
	} else {
		usage := c.Name
		if c.state != nil && len(c.state.commandPath) > 0 {
			usage = getCommandPath(c.state.commandPath)
		}
		if c.Flags != nil {
			usage += " [flags]"
		}
		if len(c.SubCommands) > 0 {
			usage += " <command>"
		}
		b.WriteString(usage)
		b.WriteRune('\n')
	}

	// Available Commands section
	if len(c.SubCommands) > 0 {
		b.WriteString("Available Commands:\n")

		sortedCommands := slices.Clone(c.SubCommands)
		slices.SortFunc(sortedCommands, func(a, b *Command) int {
			return cmp.Compare(a.Name, b.Name)
		})

		maxLen := 0
		for _, sub := range sortedCommands {
			if len(sub.Name) > maxLen {
				maxLen = len(sub.Name)
			}
		}

		for _, sub := range sortedCommands {
			if sub.ShortHelp == "" {
				fmt.Fprintf(&b, "  %s\n", sub.Name)
				continue
			}

			nameWidth := maxLen + 4
			wrapWidth := 80 - nameWidth

			lines := textutil.Wrap(sub.ShortHelp, wrapWidth)
			padding := strings.Repeat(" ", maxLen-len(sub.Name)+4)
			fmt.Fprintf(&b, "  %s%s%s\n", sub.Name, padding, lines[0])

			indentPadding := strings.Repeat(" ", nameWidth+2)
			for _, line := range lines[1:] {
				fmt.Fprintf(&b, "%s%s\n", indentPadding, line)
			}
		}
		b.WriteRune('\n')
	}

	var flags []flagInfo

	if c.state != nil && len(c.state.commandPath) > 0 {
		for i, cmd := range c.state.commandPath {
			if cmd.Flags == nil {
				continue
			}
			isGlobal := i < len(c.state.commandPath)-1
			cmd.Flags.VisitAll(func(f *flag.Flag) {
				flags = append(flags, flagInfo{
					name:   "-" + f.Name,
					usage:  f.Usage,
					defval: f.DefValue,
					global: isGlobal,
				})
			})
		}
	}

	if len(flags) > 0 {
		slices.SortFunc(flags, func(a, b flagInfo) int {
			return cmp.Compare(a.name, b.name)
		})

		maxLen := 0
		for _, f := range flags {
			if len(f.name) > maxLen {
				maxLen = len(f.name)
			}
		}

		hasLocal := false
		hasGlobal := false
		for _, f := range flags {
			if f.global {
				hasGlobal = true
			} else {
				hasLocal = true
			}
		}

		if hasLocal {
			b.WriteString("Flags:\n")
			writeFlagSection(&b, flags, maxLen, false)
			b.WriteRune('\n')
		}

		if hasGlobal {
			b.WriteString("Global Flags:\n")
			writeFlagSection(&b, flags, maxLen, true)
			b.WriteRune('\n')
		}
	}

	// Help suggestion for subcommands
	if len(c.SubCommands) > 0 {
		fmt.Fprintf(&b, "Use \"%s [command] --help\" for more information about a command.\n",
			getCommandPath(c.state.commandPath))
	}

	return strings.TrimRight(b.String(), "\n")
}

// writeFlagSection writes either the local or global flags section
func writeFlagSection(b *strings.Builder, flags []flagInfo, maxLen int, global bool) {
	for _, f := range flags {
		if f.global == global {
			nameWidth := maxLen + 4
			wrapWidth := 80 - nameWidth

			usageText := f.usage
			if f.defval != "" && f.defval != "false" {
				usageText += fmt.Sprintf(" (default %s)", f.defval)
			}

			lines := textutil.Wrap(usageText, wrapWidth)
			padding := strings.Repeat(" ", maxLen-len(f.name)+4)
			fmt.Fprintf(b, "  %s%s%s\n", f.name, padding, lines[0])

			indentPadding := strings.Repeat(" ", nameWidth+2)
			for _, line := range lines[1:] {
				fmt.Fprintf(b, "%s%s\n", indentPadding, line)
			}
		}
	}
}

type flagInfo struct {
	name   string
	usage  string
	defval string
	global bool
}

func formatFlagName(name string) string {
	return "-" + name
}

func getCommandPath(commands []*Command) string {
	var commandPath []string
	for _, c := range commands {
		commandPath = append(commandPath, c.Name)
	}
	return strings.Join(commandPath, " ")
}
