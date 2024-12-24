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

	t.Run("parse and run", func(t *testing.T) {
		t.Parallel()
		var count int

		root := &Command{
			Name:  "count",
			Usage: "count [flags] [command]",
			Flags: FlagsFunc(func(fset *flag.FlagSet) {
				fset.Bool("dry-run", false, "dry run")
			}),
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
			Exec: func(ctx context.Context, s *State) error {
				if GetFlag[bool](s, "dry-run") {
					return nil
				}
				count++
				return nil
			},
		}

		output := bytes.NewBuffer(nil)
		err := ParseAndRun(context.Background(), root, []string{"version"}, &RunOptions{
			Stdout: output,
		})
		require.NoError(t, err)
		require.Equal(t, "1.0.0\n", output.String())
		output.Reset()

		// Run the command 3 times
		for i := 0; i < 3; i++ {
			err := ParseAndRun(context.Background(), root, nil, nil)
			require.NoError(t, err)
		}
		require.Equal(t, 3, count)
		// Run with dry-run flag
		err = ParseAndRun(context.Background(), root, []string{"--dry-run"}, nil)
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

		err := ParseAndRun(context.Background(), root, []string{"verzion"}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), `unknown command "verzion". Did you mean one of these?`)
		require.Contains(t, err.Error(), `	version`)
	})
}
