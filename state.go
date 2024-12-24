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

	// TODO(mf): remove flags in favor of tracking the selected *Command
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
	// TODO(mf): we should have a way to get the selected command here to improve error messages
	if f := s.flags.Lookup(name); f != nil {
		if getter, ok := f.Value.(flag.Getter); ok {
			value := getter.Get()
			if v, ok := value.(T); ok {
				return v
			}
			msg := fmt.Sprintf("internal error: type mismatch for flag %q: registered %T, requested %T", name, value, *new(T))
			// Flag exists but type doesn't match - this is an internal error
			panic(msg)
		}
	}
	// If not found and we have a parent, try parent's flags
	if s.parent != nil {
		return GetFlag[T](s.parent, name)
	}
	// If flag not found anywhere in hierarchy, panic with helpful message
	msg := fmt.Sprintf("internal error: flag not found: %q in %s flag set", name, s.flags.Name())
	panic(msg)
}
