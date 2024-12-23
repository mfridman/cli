package cli

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFlag(t *testing.T) {
	t.Parallel()

	st := &State{
		flags: flag.NewFlagSet("root", flag.ContinueOnError),
	}
	// Capture the panic
	defer func() {
		r := recover()
		require.NotNil(t, r)
		assert.Equal(t, `flag not found: "version" in root flag set, consider filing an issue with the cli author`, r)
	}()
	// Panic because author forgot to define the flag and tried to access it. This is a programming
	// error and should be caught early
	_ = GetFlag[string](st, "version")
}
