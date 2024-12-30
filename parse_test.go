package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testState is a helper struct to hold the commands for testing
//
//		 root --verbose --version
//		 ├── add --dry-run
//		 └── nested --force
//			└── sub --echo
//	     └── hello --mandatory-flag=false --another-mandatory-flag some-value
type testState struct {
	add                *Command
	nested, sub, hello *Command
	root               *Command
}

func newTestState() testState {
	exec := func(ctx context.Context, s *State) error { return errors.New("not implemented") }
	add := &Command{
		Name: "add",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.Bool("dry-run", false, "enable dry-run mode")
		}),
		Exec: exec,
	}
	sub := &Command{
		Name: "sub",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.String("echo", "", "echo the message")
		}),
		FlagsMetadata: []FlagMetadata{
			{Name: "echo", Required: false}, // not required
		},
		Exec: exec,
	}
	hello := &Command{
		Name: "hello",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.Bool("mandatory-flag", false, "mandatory flag")
			fset.String("another-mandatory-flag", "", "another mandatory flag")
		}),
		FlagsMetadata: []FlagMetadata{
			{Name: "mandatory-flag", Required: true},
			{Name: "another-mandatory-flag", Required: true},
		},
		Exec: exec,
	}
	nested := &Command{
		Name: "nested",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.Bool("force", false, "force the operation")
		}),
		SubCommands: []*Command{sub, hello},
		Exec:        exec,
	}
	root := &Command{
		Name: "todo",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.Bool("verbose", false, "enable verbose mode")
			fset.Bool("version", false, "show version")
		}),
		SubCommands: []*Command{add, nested},
		Exec:        exec,
	}
	return testState{
		add:    add,
		nested: nested,
		sub:    sub,
		root:   root,
		hello:  hello,
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("error on parse with no exec", func(t *testing.T) {
		t.Parallel()
		cmd := &Command{
			Name: "foo",
			Exec: func(ctx context.Context, s *State) error { return nil },
			SubCommands: []*Command{
				{Name: "bar"},
			},
		}
		err := Parse(cmd, []string{"bar"})
		require.Error(t, err)
		var noExecErr *NoExecError
		require.ErrorAs(t, err, &noExecErr)
		assert.ErrorContains(t, err, `command "foo bar" has no execution function`)
	})
	t.Run("parsing errors", func(t *testing.T) {
		t.Parallel()

		err := Parse(nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "command is nil")

		err = Parse(&Command{}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "root command has no name")
	})
	t.Run("subcommand nil flags", func(t *testing.T) {
		t.Parallel()

		err := Parse(&Command{
			Name: "root",
			SubCommands: []*Command{{
				Name: "sub",
				Exec: func(ctx context.Context, s *State) error { return nil },
			}},
			Exec: func(ctx context.Context, s *State) error { return nil },
		}, []string{"sub"})
		require.NoError(t, err)
	})
	t.Run("default flag usage", func(t *testing.T) {
		t.Parallel()

		by := bytes.NewBuffer(nil)
		err := Parse(&Command{
			Name:  "root",
			Usage: "root [flags]",
			Flags: FlagsFunc(func(fset *flag.FlagSet) {
				fset.SetOutput(by)
			}),
		}, []string{"--help"})
		require.Error(t, err)
		require.ErrorIs(t, err, flag.ErrHelp)
		require.Contains(t, by.String(), "Usage:")
		require.Contains(t, by.String(), "root [flags]")
	})
	t.Run("no flags", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "item1"})
		require.NoError(t, err)
		require.NotNil(t, s.root.state)
		require.NotEmpty(t, s.root.state.commandPath)
		cmd, state := s.root.terminal()
		require.Equal(t, s.add, cmd)
		require.False(t, GetFlag[bool](state, "dry-run"))
	})
	t.Run("unknown flag", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "--unknown", "item1"})
		require.Error(t, err)
		require.Contains(t, err.Error(), `command "add": flag provided but not defined: -unknown`)
	})
	t.Run("with subcommand flags", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "--dry-run", "item1"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.add, cmd)
		assert.True(t, GetFlag[bool](state, "dry-run"))
	})
	t.Run("help flag", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--help"})
		require.Error(t, err)
		require.ErrorIs(t, err, flag.ErrHelp)
	})
	t.Run("help flag with subcommand", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "--help"})
		require.Error(t, err)
		require.ErrorIs(t, err, flag.ErrHelp)
	})
	t.Run("help flag with subcommand at s.root", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--help", "add"})
		require.Error(t, err)
		require.ErrorIs(t, err, flag.ErrHelp)
	})
	t.Run("help flag with subcommand and other flags", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "--help", "--dry-run"})
		require.Error(t, err)
		require.ErrorIs(t, err, flag.ErrHelp)
	})
	t.Run("unknown subcommand", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"unknown"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown command")
	})
	t.Run("flags at multiple levels", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "--dry-run", "item1", "--verbose"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.add, cmd)
		assert.True(t, GetFlag[bool](state, "dry-run"))
		assert.True(t, GetFlag[bool](state, "verbose"))
	})
	t.Run("nested subcommand and root flag", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--verbose", "nested", "sub", "--echo", "hello"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.sub, cmd)
		assert.Equal(t, "hello", GetFlag[string](state, "echo"))
		assert.True(t, GetFlag[bool](state, "verbose"))
	})
	t.Run("nested subcommand with mixed flags", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--echo", "hello", "--verbose"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.sub, cmd)
		assert.Equal(t, "hello", GetFlag[string](state, "echo"))
		assert.True(t, GetFlag[bool](state, "verbose"))
	})
	t.Run("end of options delimiter", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--verbose", "--", "nested", "sub", "--echo", "hello"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.root, cmd)
		assert.Equal(t, []string{"nested", "sub", "--echo", "hello"}, state.Args)
		assert.True(t, GetFlag[bool](state, "verbose"))
	})
	t.Run("flags and args", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "item1", "--dry-run", "item2"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.add, cmd)
		assert.True(t, GetFlag[bool](state, "dry-run"))
		assert.Equal(t, []string{"item1", "item2"}, state.Args)
	})
	t.Run("nested subcommand with flags and args", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--echo", "hello", "world"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.sub, cmd)
		assert.Equal(t, "hello", GetFlag[string](state, "echo"))
		assert.Equal(t, []string{"world"}, state.Args)
	})
	t.Run("subcommand flags not available in parent", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--dry-run"})
		require.Error(t, err)
		require.ErrorContains(t, err, "flag provided but not defined")
	})
	t.Run("parent flags inherited in subcommand", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--force"})
		require.NoError(t, err)
		cmd, state := s.root.terminal()
		assert.Equal(t, s.sub, cmd)
		assert.True(t, GetFlag[bool](state, "force"))
	})
	t.Run("unrelated subcommand flags not inherited in other subcommands", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--dry-run"})
		require.Error(t, err)
		require.ErrorContains(t, err, "flag provided but not defined")
	})
	t.Run("empty name in subcommand", func(t *testing.T) {
		t.Parallel()
		s := newTestState()
		s.sub.Name = ""

		err := Parse(s.root, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, `subcommand in path "todo nested" has no name`)
	})
	t.Run("required flag", func(t *testing.T) {
		t.Parallel()
		{
			s := newTestState()
			err := Parse(s.root, []string{"nested", "hello"})
			require.Error(t, err)
			require.ErrorContains(t, err, `command "todo nested hello": required flags "-mandatory-flag, -another-mandatory-flag" not set`)
		}
		{
			// Correct type - true
			s := newTestState()
			err := Parse(s.root, []string{"nested", "hello", "--mandatory-flag=true", "--another-mandatory-flag", "some-value"})
			require.NoError(t, err)
			cmd, state := s.root.terminal()
			assert.Equal(t, s.hello, cmd)
			require.True(t, GetFlag[bool](state, "mandatory-flag"))
		}
		{
			// Correct type - false
			s := newTestState()
			err := Parse(s.root, []string{"nested", "hello", "--mandatory-flag=false", "--another-mandatory-flag=some-value"})
			require.NoError(t, err)
			cmd, state := s.root.terminal()
			assert.Equal(t, s.hello, cmd)
			require.False(t, GetFlag[bool](state, "mandatory-flag"))
		}
		{
			// Incorrect type
			s := newTestState()
			err := Parse(s.root, []string{"nested", "hello", "--mandatory-flag=not-a-bool"})
			require.Error(t, err)
			require.ErrorContains(t, err, `command "hello": invalid boolean value "not-a-bool" for -mandatory-flag: parse error`)
		}
	})
	t.Run("unknown required flag set by cli author", func(t *testing.T) {
		t.Parallel()
		cmd := &Command{
			Name: "root",
			FlagsMetadata: []FlagMetadata{
				{Name: "some-other-flag", Required: true},
			},
		}
		err := Parse(cmd, nil)
		require.Error(t, err)
		// TODO(mf): consider improving this error message so it's obvious that a "required" flag
		// was set by the cli author but not registered in the flag set
		require.ErrorContains(t, err, `command "root": internal error: required flag -some-other-flag not found in flag set`)
	})
	t.Run("space in command name", func(t *testing.T) {
		t.Parallel()
		cmd := &Command{
			Name: "root",
			SubCommands: []*Command{
				{Name: "sub command"},
			},
		}
		err := Parse(cmd, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, `command name "sub command" contains spaces, must be a single word`)
	})
}
