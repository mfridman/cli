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
		st := &State{
			flags: flag.NewFlagSet("root", flag.ContinueOnError),
		}
		// Capture the panic
		defer func() {
			r := recover()
			require.NotNil(t, r)
			assert.Equal(t, `internal error: flag not found: "version" in root flag set`, r)
		}()
		// Panic because author forgot to define the flag and tried to access it. This is a
		// programming error and should be caught early
		_ = GetFlag[string](st, "version")
	})
	t.Run("flag type mismatch", func(t *testing.T) {
		st := &State{
			flags: flag.NewFlagSet("root", flag.ContinueOnError),
		}
		st.flags.String("version", "1.0", "version")
		defer func() {
			r := recover()
			require.NotNil(t, r)
			assert.Equal(t, `internal error: type mismatch for flag "version": registered string, requested int`, r)
		}()
		_ = GetFlag[int](st, "version")
	})
}
