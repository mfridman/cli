package cli

import (
	"errors"
	"flag"
	"fmt"
	"slices"
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
	if err := validateCommands(root, nil); err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Initialize or update root state
	if root.state == nil {
		root.state = &State{
			commandPath: []*Command{root},
		}
	} else {
		// Reset command path but preserve other state
		root.state.commandPath = []*Command{root}
	}
	// First split args at the -- delimiter if present
	var argsToParse []string
	var remainingArgs []string
	for i, arg := range args {
		if arg == "--" {
			argsToParse = args[:i]
			remainingArgs = args[i+1:]
			break
		}
	}
	if argsToParse == nil {
		argsToParse = args
	}

	current := root
	var commandChain []*Command
	commandChain = append(commandChain, root)

	// Create combined flags with all parent flags
	combinedFlags := flag.NewFlagSet(root.Name, flag.ContinueOnError)

	// First pass: process commands and build the flag set
	for _, arg := range argsToParse {
		// Skip anything that looks like a flag
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// Try to traverse to subcommand
		if len(current.SubCommands) > 0 {
			if sub := current.findSubCommand(arg); sub != nil {
				// Update root state's command path
				root.state.commandPath = append(slices.Clone(root.state.commandPath), sub)

				if sub.Flags == nil {
					sub.Flags = flag.NewFlagSet(sub.Name, flag.ContinueOnError)
				}
				current = sub
				commandChain = append(commandChain, sub)
				continue
			}
			return current.formatUnknownCommandError(arg)
		}
		break
	}

	// Add the help check here, after we've found the correct command
	for _, arg := range argsToParse {
		if arg == "-h" || arg == "--h" || arg == "-help" || arg == "--help" {
			combinedFlags.Usage = func() { _ = current.showHelp() }
			_ = current.showHelp()
			return flag.ErrHelp
		}
	}

	// Add flags in reverse order for proper precedence
	for i := len(commandChain) - 1; i >= 0; i-- {
		cmd := commandChain[i]
		if cmd.Flags != nil {
			cmd.Flags.VisitAll(func(f *flag.Flag) {
				if combinedFlags.Lookup(f.Name) == nil {
					combinedFlags.Var(f.Value, f.Name, f.Usage)
				}
			})
		}
	}

	// Let ParseToEnd handle the flag parsing
	if err := xflag.ParseToEnd(combinedFlags, argsToParse); err != nil {
		return fmt.Errorf("command %q: %w", current.Name, err)
	}

	// Check required flags
	var missingFlags []string
	for _, cmd := range commandChain {
		if len(cmd.FlagsMetadata) > 0 {
			for _, flagMetadata := range cmd.FlagsMetadata {
				if !flagMetadata.Required {
					continue
				}
				flag := combinedFlags.Lookup(flagMetadata.Name)
				if flag == nil {
					return fmt.Errorf("command %q: internal error: required flag %s not found in flag set", getCommandPath(root.state.commandPath), formatFlagName(flagMetadata.Name))
				}
				if flag.Value.String() == flag.DefValue {
					missingFlags = append(missingFlags, formatFlagName(flagMetadata.Name))
				}
			}
		}
	}
	if len(missingFlags) > 0 {
		msg := "required flag"
		if len(missingFlags) > 1 {
			msg += "s"
		}
		return fmt.Errorf("command %q: %s %q not set", getCommandPath(root.state.commandPath), msg, strings.Join(missingFlags, ", "))
	}

	// Skip past command names in remaining args
	parsed := combinedFlags.Args()
	startIdx := 0
	for _, arg := range parsed {
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

	// Combine remaining parsed args and everything after delimiter
	var finalArgs []string
	if startIdx < len(parsed) {
		finalArgs = append(finalArgs, parsed[startIdx:]...)
	}
	if len(remainingArgs) > 0 {
		finalArgs = append(finalArgs, remainingArgs...)
	}
	root.state.Args = finalArgs

	if current.Exec == nil {
		return &NoExecError{
			Command: root, // Pass the root command which has the state with the full path
		}
	}
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
		return fmt.Errorf("command name %q contains spaces, must be a single word", root.Name)
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
