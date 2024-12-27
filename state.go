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

	// The full name of the command, including parent commands. E.g., "cli todo list all"
	fullName string
	// Reference to the command this state belongs to
	cmd    *Command
	parent *State
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
	if f := s.cmd.Flags.Lookup(name); f != nil {
		if getter, ok := f.Value.(flag.Getter); ok {
			value := getter.Get()
			if v, ok := value.(T); ok {
				return v
			}
			msg := fmt.Sprintf("internal error: type mismatch for flag %q in command %q: registered %T, requested %T", formatFlagName(name), s.fullName, value, *new(T))
			// Flag exists but type doesn't match - this is an internal error
			panic(msg)
		}
	}
	// If not found and we have a parent, try parent's flags
	if s.parent != nil {
		return GetFlag[T](s.parent, name)
	}
	// If flag not found anywhere in hierarchy, panic with helpful message
	msg := fmt.Sprintf("internal error: flag %q not found in %q flag set", formatFlagName(name), s.fullName)
	panic(msg)
}

func formatFlagName(name string) string {
	return "-" + name
}
