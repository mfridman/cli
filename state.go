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

	flags  *flag.FlagSet
	parent *State
}

// GetFlag retrieves a flag value by name, with type inference. It traverses up the state hierarchy
// to find the flag, allowing access to parent command flags. Example usage:
//
//	verbose := GetFlag[bool](state, "verbose")
//	count := GetFlag[int](state, "count")
//	path := GetFlag[string](state, "path")
//
// If the flag isn't found, it panics with a detailed error message.
//
// Why panic? Because if a flag is missing, it's likely a programming error or a missing flag
// definition, and it's better to fail LOUD and EARLY than to silently ignore the issue and cause
// unexpected behavior.
func GetFlag[T any](s *State, name string) T {
	// Try to get flag from current command's flags
	if f := s.flags.Lookup(name); f != nil {
		if v, ok := f.Value.(flag.Getter).Get().(T); ok {
			return v
		}
	}
	// If not found and we have a parent, try parent's flags
	if s.parent != nil {
		return GetFlag[T](s.parent, name)
	}
	// If flag not found anywhere in hierarchy, panic with helpful message
	panic(fmt.Sprintf("flag not found: %q in %s flag set, consider filing an issue with the cli author", name, s.flags.Name()))
}
