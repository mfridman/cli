package cli

import (
	"bytes"
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("print version", func(t *testing.T) {
		t.Parallel()

		root := &Command{
			Name:  "printer",
			Usage: "printer [flags] [command]",
			SubCommands: []*Command{
				{
					Name:  "version",
					Usage: "show version",
					Exec: func(ctx context.Context, s *State) error {
						_, _ = s.Stdout.Write([]byte("1.0.0\n"))
						return nil
					},
				},
			},
			Exec: func(ctx context.Context, s *State) error { return nil },
		}
		err := Parse(root, []string{"version"})
		require.NoError(t, err)

		output := bytes.NewBuffer(nil)
		require.NoError(t, err)
		err = Run(context.Background(), root, &RunOptions{Stdout: output})
		require.NoError(t, err)
		require.Equal(t, "1.0.0\n", output.String())
	})

	t.Run("parse and run", func(t *testing.T) {
		t.Parallel()
		var count int

		root := &Command{
			Name:  "count",
			Usage: "count [flags] [command]",
			Flags: FlagsFunc(func(f *flag.FlagSet) {
				f.Bool("dry-run", false, "dry run")
			}),
			Exec: func(ctx context.Context, s *State) error {
				if !GetFlag[bool](s, "dry-run") {
					count++
				}
				return nil
			},
		}
		err := Parse(root, nil)
		require.NoError(t, err)
		// Run the command 3 times
		for i := 0; i < 3; i++ {
			err := Run(context.Background(), root, nil)
			require.NoError(t, err)
		}
		require.Equal(t, 3, count)
		// Run with dry-run flag
		err = Parse(root, []string{"--dry-run"})
		require.NoError(t, err)
		err = Run(context.Background(), root, nil)
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})
	t.Run("typo suggestion", func(t *testing.T) {
		t.Parallel()

		root := &Command{
			Name:  "count",
			Usage: "count [flags] [command]",
			SubCommands: []*Command{
				{
					Name:  "version",
					Usage: "show version",
					Exec: func(ctx context.Context, s *State) error {
						_, _ = s.Stdout.Write([]byte("1.0.0\n"))
						return nil
					},
				},
			},
			Exec: func(ctx context.Context, s *State) error { return nil },
		}

		err := Parse(root, []string{"verzion"})
		require.Error(t, err)
		require.Contains(t, err.Error(), `unknown command "verzion". Did you mean one of these?`)
		require.Contains(t, err.Error(), `	version`)
	})
}
