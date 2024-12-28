package cli

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFlag(t *testing.T) {
	t.Parallel()

	t.Run("flag not found", func(t *testing.T) {
		cmd := &Command{
			Name:  "root",
			Flags: flag.NewFlagSet("root", flag.ContinueOnError),
		}
		state := &State{
			commandPath: []*Command{cmd},
		}
		defer func() {
			r := recover()
			require.NotNil(t, r)
			assert.Equal(t, `internal error: flag "-version" not found in "root" flag set`, r)
		}()
		// Panic because author tried to access a flag that doesn't exist in any of the commands
		_ = GetFlag[string](state, "version")
	})
	t.Run("flag type mismatch", func(t *testing.T) {
		cmd := &Command{
			Name:  "root",
			Flags: FlagsFunc(func(f *flag.FlagSet) { f.String("version", "1.0.0", "show version") }),
		}
		state := &State{
			commandPath: []*Command{cmd},
		}
		defer func() {
			r := recover()
			require.NotNil(t, r)
			assert.Equal(t, `internal error: type mismatch for flag "-version" in command "root": registered string, requested int`, r)
		}()
		// Panic because author tried to access a registered flag with the wrong type
		_ = GetFlag[int](state, "version")
	})
}
