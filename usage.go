package cli

import (
	"cmp"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/mfridman/cli/pkg/textutil"
)

func DefaultUsage(c *Command) string {
	if c == nil {
		return ""
	}

	// Get terminal command from state
	terminalCmd, _ := c.terminal()

	var b strings.Builder

	if terminalCmd.UsageFunc != nil {
		return terminalCmd.UsageFunc(terminalCmd)
	}

	if terminalCmd.ShortHelp != "" {
		b.WriteString(terminalCmd.ShortHelp)
		b.WriteString("\n\n")
	}

	b.WriteString("Usage:\n")
	if terminalCmd.Usage != "" {
		b.WriteString("  " + terminalCmd.Usage + "\n")
	} else {
		usage := terminalCmd.Name
		if c.state != nil && len(c.state.commandPath) > 0 {
			usage = getCommandPath(c.state.commandPath)
		}
		if terminalCmd.Flags != nil {
			usage += " [flags]"
		}
		if len(terminalCmd.SubCommands) > 0 {
			usage += " <command>"
		}
		b.WriteString("  " + usage + "\n")
	}
	b.WriteString("\n")

	if len(terminalCmd.SubCommands) > 0 {
		b.WriteString("Available Commands:\n")
		sortedCommands := slices.Clone(terminalCmd.SubCommands)
		slices.SortFunc(sortedCommands, func(a, b *Command) int {
			return cmp.Compare(a.Name, b.Name)
		})

		maxNameLen := 0
		for _, sub := range sortedCommands {
			if len(sub.Name) > maxNameLen {
				maxNameLen = len(sub.Name)
			}
		}

		nameWidth := maxNameLen + 4
		wrapWidth := 80 - nameWidth

		for _, sub := range sortedCommands {
			if sub.ShortHelp == "" {
				fmt.Fprintf(&b, "  %s\n", sub.Name)
				continue
			}

			lines := textutil.Wrap(sub.ShortHelp, wrapWidth)
			padding := strings.Repeat(" ", maxNameLen-len(sub.Name)+4)
			fmt.Fprintf(&b, "  %s%s%s\n", sub.Name, padding, lines[0])

			indentPadding := strings.Repeat(" ", nameWidth+2)
			for _, line := range lines[1:] {
				fmt.Fprintf(&b, "%s%s\n", indentPadding, line)
			}
		}
		b.WriteString("\n")
	}

	var flags []flagInfo
	if c.state != nil && len(c.state.commandPath) > 0 {
		for i, cmd := range c.state.commandPath {
			if cmd.Flags == nil {
				continue
			}
			isGlobal := i < len(c.state.commandPath)-1
			cmd.Flags.VisitAll(func(f *flag.Flag) {
				flags = append(flags, flagInfo{
					name:   "-" + f.Name,
					usage:  f.Usage,
					defval: f.DefValue,
					global: isGlobal,
				})
			})
		}
	}

	if len(flags) > 0 {
		slices.SortFunc(flags, func(a, b flagInfo) int {
			return cmp.Compare(a.name, b.name)
		})

		maxFlagLen := 0
		for _, f := range flags {
			if len(f.name) > maxFlagLen {
				maxFlagLen = len(f.name)
			}
		}

		hasLocal := false
		hasGlobal := false
		for _, f := range flags {
			if f.global {
				hasGlobal = true
			} else {
				hasLocal = true
			}
		}

		if hasLocal {
			b.WriteString("Flags:\n")
			writeFlagSection(&b, flags, maxFlagLen, false)
			b.WriteString("\n")
		}

		if hasGlobal {
			b.WriteString("Global Flags:\n")
			writeFlagSection(&b, flags, maxFlagLen, true)
			b.WriteString("\n")
		}
	}

	if len(terminalCmd.SubCommands) > 0 {
		cmdName := terminalCmd.Name
		if c.state != nil && len(c.state.commandPath) > 0 {
			cmdName = getCommandPath(c.state.commandPath)
		}
		fmt.Fprintf(&b, "Use \"%s [command] --help\" for more information about a command.\n", cmdName)
	}

	return strings.TrimRight(b.String(), "\n")
}

// writeFlagSection handles the formatting of flag descriptions
func writeFlagSection(b *strings.Builder, flags []flagInfo, maxLen int, global bool) {
	nameWidth := maxLen + 4
	wrapWidth := 80 - nameWidth

	for _, f := range flags {
		if f.global != global {
			continue
		}

		description := f.usage
		if f.defval != "" {
			description += fmt.Sprintf(" (default: %s)", f.defval)
		}

		lines := textutil.Wrap(description, wrapWidth)
		padding := strings.Repeat(" ", maxLen-len(f.name)+4)
		fmt.Fprintf(b, "  %s%s%s\n", f.name, padding, lines[0])

		indentPadding := strings.Repeat(" ", nameWidth+2)
		for _, line := range lines[1:] {
			fmt.Fprintf(b, "%s%s\n", indentPadding, line)
		}
	}
}

type flagInfo struct {
	name   string
	usage  string
	defval string
	global bool
}
