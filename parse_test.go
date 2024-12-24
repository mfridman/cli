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
//	 root --verbose --version
//	 ├── add --dry-run
//	 └── nested --force
//		└── sub --echo
type testState struct {
	add         *Command
	nested, sub *Command
	root        *Command
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
		Exec: exec,
	}
	nested := &Command{
		Name: "nested",
		Flags: FlagsFunc(func(fset *flag.FlagSet) {
			fset.Bool("force", false, "force the operation")
		}),
		SubCommands: []*Command{sub},
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
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("no error on parse with no exec", func(t *testing.T) {
		t.Parallel()

		err := Parse(&Command{Name: "root"}, nil)
		require.NoError(t, err)
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
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.add, s.root.selected)
		require.False(t, GetFlag[bool](s.root.selected.state, "dry-run"))
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
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.add, s.root.selected)
		require.True(t, GetFlag[bool](s.root.selected.state, "dry-run"))
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
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.add, s.root.selected)
		require.True(t, GetFlag[bool](s.root.selected.state, "dry-run"))
		require.True(t, GetFlag[bool](s.root.selected.state, "verbose"))
	})
	t.Run("nested subcommand with s.root flag", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--verbose", "nested", "sub", "--echo", "hello"})
		require.NoError(t, err)
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.sub, s.root.selected)
		require.Equal(t, "hello", GetFlag[string](s.root.selected.state, "echo"))
		require.True(t, GetFlag[bool](s.root.selected.state, "verbose"))
	})
	t.Run("nested subcommand with mixed flags", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--echo", "hello", "--verbose"})
		require.NoError(t, err)
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.sub, s.root.selected)
		require.Equal(t, "hello", GetFlag[string](s.root.selected.state, "echo"))
		require.True(t, GetFlag[bool](s.root.selected.state, "verbose"))
	})
	t.Run("end of options delimiter", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"--verbose", "--", "nested", "sub", "--echo", "hello"})
		require.NoError(t, err)
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.root.Name, s.root.selected.Name)
		require.True(t, GetFlag[bool](s.root.selected.state, "verbose"))
		assert.Equal(t, []string{"nested", "sub", "--echo", "hello"}, s.root.selected.state.Args)
	})
	t.Run("flags and args", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"add", "item1", "--dry-run", "item2"})
		require.NoError(t, err)
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.add, s.root.selected)
		require.True(t, GetFlag[bool](s.root.selected.state, "dry-run"))
		assert.Equal(t, []string{"item1", "item2"}, s.root.selected.state.Args)
	})
	t.Run("nested subcommand with flags and args", func(t *testing.T) {
		t.Parallel()
		s := newTestState()

		err := Parse(s.root, []string{"nested", "sub", "--echo", "hello", "world"})
		require.NoError(t, err)
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		require.Equal(t, s.sub, s.root.selected)
		require.Equal(t, "hello", GetFlag[string](s.root.selected.state, "echo"))
		assert.Equal(t, []string{"world"}, s.root.selected.state.Args)
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
		require.NotNil(t, s.root.selected)
		require.NotNil(t, s.root.selected.state)
		assert.True(t, GetFlag[bool](s.root.selected.state, "force"))
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
}
