package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// ParseAndRun parses the command hierarchy and runs the command. A convenience function that
// combines [Parse] and [Run] into a single call. See [Parse] and [Run] for more details.
func ParseAndRun(
	ctx context.Context,
	root *Command,
	args []string,
	options *RunOptions,
) error {
	if err := Parse(root, args); err != nil {
		return err
	}
	return Run(ctx, root, options)
}

// RunOptions specifies options for running a command.
type RunOptions struct {
	// Stdin, Stdout, and Stderr are the standard input, output, and error streams for the command.
	// If any of these are nil, the command will use the default streams ([os.Stdin], [os.Stdout],
	// and [os.Stderr], respectively).
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

// Run executes the current command. It returns an error if the command has not been parsed or if
// the command has no execution function.
//
// The options parameter may be nil, in which case default values are used. See [RunOptions] for
// more details.
func Run(ctx context.Context, root *Command, options *RunOptions) error {
	if root.selected == nil {
		return errors.New("command has not been parsed")
	}
	options = checkAndSetRunOptions(options)
	updateState(root.selected.state, options)

	// If it is the root command, and it has no execution function, return an error and print help
	if root.selected == root && root.selected.Exec == nil {
		return root.selected.showHelp()
	}
	if root.selected.Exec == nil {
		return fmt.Errorf("command %q has no execution function", root.selected.Name)
	}

	if err := root.selected.Exec(ctx, root.selected.state); err != nil {
		// TODO(mf): revisit this error handling, not even sure if it's necessary
		if cliErr := (*Error)(nil); errors.As(err, &cliErr) {
			_ = root.selected.showHelp()
			return err
		}
		return err
	}
	return nil
}

func updateState(s *State, opt *RunOptions) {
	if s.Stdin == nil {
		s.Stdin = opt.Stdin
	}
	if s.Stdout == nil {
		s.Stdout = opt.Stdout
	}
	if s.Stderr == nil {
		s.Stderr = opt.Stderr
	}
}

func checkAndSetRunOptions(opt *RunOptions) *RunOptions {
	if opt == nil {
		opt = &RunOptions{}
	}
	if opt.Stdin == nil {
		opt.Stdin = os.Stdin
	}
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Stderr == nil {
		opt.Stderr = os.Stderr
	}
	return opt
}
