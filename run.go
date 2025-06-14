package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

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
	if root == nil {
		return errors.New("root command is nil")
	}
	if root.state == nil || len(root.state.path) == 0 {
		return errors.New("command not parsed")
	}
	cmd := root.terminal()
	if cmd == nil {
		// This should never happen, but if it does, it's likely a bug in the Parse function.
		return errors.New("no terminal command found")
	}

	options = checkAndSetRunOptions(options)
	updateState(root.state, options)

	return run(ctx, cmd, root.state)
}

func run(ctx context.Context, cmd *Command, state *State) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			switch err := r.(type) {
			case error:
				retErr = fmt.Errorf("internal: %v", err)
			default:
				retErr = fmt.Errorf("recover: %v", r)
			}
		}
	}()
	return cmd.Exec(ctx, state)
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
