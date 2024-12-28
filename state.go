package cli

import (
	"flag"
	"fmt"
	"io"
)

// State represents the shared state for a command execution. It maintains a hierarchical structure
// that allows child commands to access global flags defined in parent commands. Use [GetFlag] to
// retrieve flag values by name.
type State struct {
	// Args contains the remaining arguments after flag parsing.
	Args []string

	// Standard I/O streams.
	Stdin          io.Reader
	Stdout, Stderr io.Writer

	commandPath []*Command
}

// GetFlag retrieves a flag value by name, with type inference. It traverses up the state hierarchy
// to find the flag, allowing access to parent command flags. Example usage:
//
//	verbose := GetFlag[bool](state, "verbose")
//	count := GetFlag[int](state, "count")
//	path := GetFlag[string](state, "path")
//
// If the flag isn't known, or is the wrong type, it panics with a detailed error message.
//
// Why panic? Because if a flag is missing, it's likely a programming error or a missing flag
// definition, and it's better to fail LOUD and EARLY than to silently ignore the issue and cause
// unexpected behavior.
func GetFlag[T any](s *State, name string) T {
	// Try to find the flag in each command's flag set, starting from the current command
	for i := len(s.commandPath) - 1; i >= 0; i-- {
		cmd := s.commandPath[i]
		if cmd.Flags == nil {
			continue
		}

		if f := cmd.Flags.Lookup(name); f != nil {
			if getter, ok := f.Value.(flag.Getter); ok {
				value := getter.Get()
				if v, ok := value.(T); ok {
					return v
				}
				msg := fmt.Sprintf("internal error: type mismatch for flag %q in command %q: registered %T, requested %T",
					formatFlagName(name),
					getCommandPath(s.commandPath),
					value,
					*new(T),
				)
				// Flag exists but type doesn't match - this is an internal error
				panic(msg)
			}
		}
	}

	// If flag not found anywhere in hierarchy, panic with helpful message
	msg := fmt.Sprintf("internal error: flag %q not found in %q flag set",
		formatFlagName(name),
		getCommandPath(s.commandPath),
	)
	panic(msg)
}
