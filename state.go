package cli

import (
	"flag"
	"fmt"
	"io"
)

// State holds command information during Exec function execution, allowing child commands to access
// parent flags. Use [GetFlag] to get flag values across the command hierarchy.
type State struct {
	// Args contains the remaining arguments after flag parsing.
	Args []string

	// Standard I/O streams.
	Stdin          io.Reader
	Stdout, Stderr io.Writer

	commandPath []*Command
}

// GetFlag retrieves a flag value by name from the command hierarchy. It first checks the current
// command's flags, then walks up through parent commands.
//
// If the flag doesn't exist or if the type doesn't match the requested type T an error will be
// raised in the Run function. This is an internal error and should never happen in normal usage.
// This ensures flag-related programming errors are caught early during development.
//
//	verbose := GetFlag[bool](state, "verbose")
//	count := GetFlag[int](state, "count")
//	path := GetFlag[string](state, "path")
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
				err := fmt.Errorf("type mismatch for flag %q in command %q: registered %T, requested %T",
					formatFlagName(name),
					getCommandPath(s.commandPath),
					value,
					*new(T),
				)
				// Flag exists but type doesn't match - this is an internal error
				panic(err)
			}
		}
	}

	// If flag not found anywhere in hierarchy, panic with helpful message
	err := fmt.Errorf("flag %q not found in command %q flag set",
		formatFlagName(name),
		getCommandPath(s.commandPath),
	)
	panic(err)
}
