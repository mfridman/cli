package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/mfridman/xflag"
)

// Parse traverses the command hierarchy and parses arguments. It returns an error if parsing fails
// at any point.
//
// This function is the main entry point for parsing command-line arguments and should be called
// with the root command and the arguments to parse, typically os.Args[1:]. Once parsing is
// complete, the root command is ready to be executed with the [Run] function.
func Parse(root *Command, args []string) error {
	if root == nil {
		return fmt.Errorf("failed to parse: root command is nil")
	}
	// Validate all commands have names
	if err := validateCommands(root, nil); err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Initialize root command state and flags if needed
	if root.state == nil {
		root.state = &State{}
	}
	if root.state.flags == nil {
		if root.Flags == nil {
			root.Flags = flag.NewFlagSet(root.Name, flag.ContinueOnError)
		}
		root.state.flags = root.Flags
	}

	current := root
	current.selected = current

	// Handle "--" delimiter for separating flags from positional arguments
	delimiterPos := -1
	for i, arg := range args {
		if arg == "--" {
			delimiterPos = i
			break
		}
	}
	// If we found a delimiter, only traverse commands before it
	argsToTraverse := args
	var afterDelimiter []string
	if delimiterPos >= 0 {
		argsToTraverse = args[:delimiterPos]
		if delimiterPos+1 < len(args) {
			afterDelimiter = args[delimiterPos+1:]
		}
	}

	// Track command chain for proper arg handling, this ensures that subcommands aren't included in
	// the final argument list
	var commandChain []*Command
	commandChain = append(commandChain, root)

	// Process commands
	for len(argsToTraverse) > 0 {
		// Check for help flags at current level
		switch argsToTraverse[0] {
		case "-h", "--h", "-help", "--help":
			return current.showHelp()
		}

		// Skip flags while looking for commands
		if strings.HasPrefix(argsToTraverse[0], "-") {
			argsToTraverse = argsToTraverse[1:]
			continue
		}

		// Look for subcommand
		if sub := current.findSubCommand(argsToTraverse[0]); sub != nil {
			// Initialize subcommand state if needed
			if sub.state == nil {
				sub.state = &State{}
			}
			if sub.Flags == nil {
				sub.Flags = flag.NewFlagSet(sub.Name, flag.ContinueOnError)
			}
			sub.state.flags = sub.Flags
			sub.state.parent = current.state
			current = sub
			commandChain = append(commandChain, sub)
			argsToTraverse = argsToTraverse[1:]
		} else {
			if len(current.SubCommands) > 0 {
				return current.formatUnknownCommandError(argsToTraverse[0])
			}
			break
		}
	}

	// Store the current command for testing/reference
	root.selected = current

	// Create a combined FlagSet with strict hierarchy
	combinedFlags := flag.NewFlagSet(current.Name, flag.ContinueOnError)
	combinedFlags.SetOutput(io.Discard)
	combinedFlags.Usage = func() {
		_ = current.showHelp()
	}

	// Add flags in reverse order (current command first, then parents) This ensures proper flag
	// precedence
	for i := len(commandChain) - 1; i >= 0; i-- {
		cmd := commandChain[i]
		if cmd.Flags != nil {
			cmd.Flags.VisitAll(func(f *flag.Flag) {
				// Only add the flag if it hasn't been defined yet
				if combinedFlags.Lookup(f.Name) == nil {
					combinedFlags.Var(f.Value, f.Name, f.Usage)
				}
			})
		}
	}

	// Parse flags up to delimiter if present
	argsToParse := args
	if delimiterPos >= 0 {
		argsToParse = args[:delimiterPos]
	}

	if err := xflag.ParseToEnd(combinedFlags, argsToParse); err != nil {
		return fmt.Errorf("error in command %q: %w", current.Name, err)
	}

	// Get remaining args that weren't consumed by flag parsing
	remaining := combinedFlags.Args()

	// Find where the actual arguments start by skipping past command names
	startIdx := 0
	for _, arg := range remaining {
		// Check if this arg matches any command in our chain
		isCommand := false
		for _, cmd := range commandChain {
			if arg == cmd.Name {
				startIdx++
				isCommand = true
				break
			}
		}
		if !isCommand {
			break
		}
	}

	// Only slice if we have a valid start index
	if startIdx < len(remaining) {
		remaining = remaining[startIdx:]
	} else {
		remaining = nil
	}

	// Store remaining args plus anything after delimiter
	current.state.Args = append(remaining, afterDelimiter...)

	return nil
}

func validateCommands(root *Command, path []string) error {
	if root.Name == "" {
		if len(path) == 0 {
			return errors.New("root command has no name")
		}
		return fmt.Errorf("subcommand in path %q has no name", strings.Join(path, " "))
	}
	// Ensure name has no spaces
	if strings.Contains(root.Name, " ") {
		return fmt.Errorf("command name %q contains spaces", root.Name)
	}

	// Add current command to path for nested validation
	currentPath := append(path, root.Name)

	// Recursively validate all subcommands
	for _, sub := range root.SubCommands {
		if err := validateCommands(sub, currentPath); err != nil {
			return err
		}
	}
	return nil
}
