package cli

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"slices"
	"strings"
)

// Command represents a CLI command or subcommand within the application's command hierarchy.
type Command struct {
	// Name is always a single word representing the command's name. It is used to identify the
	// command in the command hierarchy and in help text.
	Name string

	// Usage provides the command's full usage pattern.
	//
	// Example: "cli todo list [flags]"
	Usage string

	// ShortHelp is a brief description of the command's purpose. It is displayed in the help text
	// when the command is shown.
	ShortHelp string

	// UsageFunc is an optional function that can be used to generate a custom usage string for the
	// command. It receives the current command and should return a string with the full usage
	// pattern.
	UsageFunc func(*Command) string

	// Flags holds the command-specific flag definitions. Each command maintains its own flag set
	// for parsing arguments.
	Flags *flag.FlagSet
	// RequiredFlags is a list of flag names that are required for the command to run. If any of
	// these flags are missing, the command will not execute and will show its help text instead.
	//
	// TODO(mf): maybe thise should be a proper data structure instead of a list of strings to allow
	// for more flexibility in the future.
	RequiredFlags []string

	// SubCommands is a list of nested commands that exist under this command.
	SubCommands []*Command

	// Exec defines the command's execution logic. It receives the current application [State] and
	// returns an error if execution fails. This function is called when [Run] is invoked on the
	// command.
	//
	// May return a [HelpError] to indicate that the command should display its help text when [Run]
	// is called.
	Exec func(ctx context.Context, s *State) error

	state *State
	// TODO(mf): remove this in favor of tracking the selected *Command in the state
	selected *Command
}

// FlagsFunc is a helper function that creates a new [flag.FlagSet] and applies the given function
// to it. Intended for use in command definitions to simplify flag setup. Example usage:
//
//	cmd.Flags = cli.FlagsFunc(func(f *flag.FlagSet) {
//	    f.Bool("verbose", false, "enable verbose output")
//	    f.String("output", "", "output file")
//	    f.Int("count", 0, "number of items")
//	})
func FlagsFunc(fn func(*flag.FlagSet)) *flag.FlagSet {
	fset := flag.NewFlagSet("", flag.ContinueOnError)
	fn(fset)
	return fset
}

// findSubCommand searches for a subcommand by name and returns it if found. Returns nil if no
// subcommand with the given name exists.
func (c *Command) findSubCommand(name string) *Command {
	for _, sub := range c.SubCommands {
		if strings.EqualFold(sub.Name, name) {
			return sub
		}
	}
	return nil
}

func (c *Command) showHelp() error {
	w := c.Flags.Output()

	if c.UsageFunc != nil {
		fmt.Fprintf(w, "%s\n", c.UsageFunc(c))
		return nil
	}

	// Display command description first if available, with wrapping
	if c.ShortHelp != "" {
		for _, line := range wrapText(c.ShortHelp, 80) {
			fmt.Fprintf(w, "%s\n", line)
		}
		fmt.Fprintln(w)
	}

	// Show usage pattern
	fmt.Fprintf(w, "Usage:\n  ")
	if c.Usage != "" {
		fmt.Fprintf(w, "%s\n", c.Usage)
	} else {
		usage := c.Name
		if c.Flags != nil {
			usage += " [flags]"
		}
		if len(c.SubCommands) > 0 {
			usage += " <command>"
		}
		fmt.Fprintf(w, "%s\n", usage)
	}
	fmt.Fprintln(w)

	// Show available subcommands
	if len(c.SubCommands) > 0 {
		fmt.Fprintf(w, "Available Commands:\n")

		sortedCommands := slices.Clone(c.SubCommands)
		slices.SortFunc(sortedCommands, func(a, b *Command) int {
			return cmp.Compare(a.Name, b.Name)
		})

		maxLen := 0
		for _, sub := range sortedCommands {
			if len(sub.Name) > maxLen {
				maxLen = len(sub.Name)
			}
		}

		for _, sub := range sortedCommands {
			if sub.ShortHelp == "" {
				fmt.Fprintf(w, "  %s\n", sub.Name)
				continue
			}

			nameWidth := maxLen + 4
			wrapWidth := 80 - nameWidth

			lines := wrapText(sub.ShortHelp, wrapWidth)
			padding := strings.Repeat(" ", maxLen-len(sub.Name)+4)
			fmt.Fprintf(w, "  %s%s%s\n", sub.Name, padding, lines[0])

			indentPadding := strings.Repeat(" ", nameWidth+2)
			for _, line := range lines[1:] {
				fmt.Fprintf(w, "%s%s\n", indentPadding, line)
			}
		}
		fmt.Fprintln(w)
	}

	// Collect and format all flags
	type flagInfo struct {
		name   string
		usage  string
		defval string
		global bool
	}
	var flags []flagInfo

	// Local flags
	if c.Flags != nil {
		c.Flags.VisitAll(func(f *flag.Flag) {
			flags = append(flags, flagInfo{
				name:   "-" + f.Name,
				usage:  f.Usage,
				defval: f.DefValue,
				global: false,
			})
		})
	}

	// Global flags
	if c.state.parent != nil {
		p := c.state.parent
		for p != nil {
			if p.flags != nil {
				p.flags.VisitAll(func(f *flag.Flag) {
					flags = append(flags, flagInfo{
						name:   "-" + f.Name,
						usage:  f.Usage,
						defval: f.DefValue,
						global: true,
					})
				})
			}
			p = p.parent
		}
	}

	if len(flags) > 0 {
		// Sort flags by name
		slices.SortFunc(flags, func(a, b flagInfo) int {
			return cmp.Compare(a.name, b.name)
		})

		// Find the longest flag name for alignment
		maxLen := 0
		for _, f := range flags {
			if len(f.name) > maxLen {
				maxLen = len(f.name)
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

		// Print local flags first
		if hasLocal {
			fmt.Fprintf(w, "Flags:\n")
			for _, f := range flags {
				if !f.global {
					nameWidth := maxLen + 4
					wrapWidth := 80 - nameWidth

					// Prepare the usage text with default value if needed
					usageText := f.usage
					if f.defval != "" && f.defval != "false" {
						usageText += fmt.Sprintf(" (default %s)", f.defval)
					}

					// Wrap the usage text
					lines := wrapText(usageText, wrapWidth)

					// Print first line with flag name
					padding := strings.Repeat(" ", maxLen-len(f.name)+4)
					fmt.Fprintf(w, "  %s%s%s\n", f.name, padding, lines[0])

					// Print subsequent lines with proper padding
					indentPadding := strings.Repeat(" ", nameWidth+2)
					for _, line := range lines[1:] {
						fmt.Fprintf(w, "%s%s\n", indentPadding, line)
					}
				}
			}
			fmt.Fprintln(w)
		}

		// Then print global flags
		if hasGlobal {
			fmt.Fprintf(w, "Global Flags:\n")
			for _, f := range flags {
				if f.global {
					nameWidth := maxLen + 4
					wrapWidth := 80 - nameWidth

					// Prepare the usage text with default value if needed
					usageText := f.usage
					if f.defval != "" && f.defval != "false" {
						usageText += fmt.Sprintf(" (default %s)", f.defval)
					}

					// Wrap the usage text
					lines := wrapText(usageText, wrapWidth)

					// Print first line with flag name
					padding := strings.Repeat(" ", maxLen-len(f.name)+4)
					fmt.Fprintf(w, "  %s%s%s\n", f.name, padding, lines[0])

					// Print subsequent lines with proper padding
					indentPadding := strings.Repeat(" ", nameWidth+2)
					for _, line := range lines[1:] {
						fmt.Fprintf(w, "%s%s\n", indentPadding, line)
					}
				}
			}
			fmt.Fprintln(w)
		}
	}

	// Show help hint for subcommands
	if len(c.SubCommands) > 0 {
		fmt.Fprintf(w, "Use \"%s [command] --help\" for more information about a command.\n", c.Name)
	}

	return flag.ErrHelp
}

func (c *Command) getSuggestions(unknownCmd string) []string {
	var availableCommands []string
	for _, subcmd := range c.SubCommands {
		availableCommands = append(availableCommands, subcmd.Name)
	}

	suggestions := make([]struct {
		name  string
		score float64
	}, 0, len(availableCommands))

	// Calculate similarity scores
	for _, name := range availableCommands {
		score := calculateSimilarity(unknownCmd, name)
		if score > 0.5 { // Only include reasonably similar commands
			suggestions = append(suggestions, struct {
				name  string
				score float64
			}{name, score})
		}
	}
	// Sort suggestions by score (highest first)
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].score > suggestions[i].score {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
	// Get top 3 suggestions
	maxSuggestions := 3
	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].name)
	}

	return result
}

func (c *Command) formatUnknownCommandError(unknownCmd string) error {
	suggestions := c.getSuggestions(unknownCmd)
	if len(suggestions) > 0 {
		return fmt.Errorf("unknown command %q\nDid you mean one of these?\n\t%s",
			unknownCmd,
			strings.Join(suggestions, "\n\t"))
	}
	return fmt.Errorf("unknown command %q", unknownCmd)
}

func calculateSimilarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Perfect match
	if a == b {
		return 1.0
	}
	// Prefix match bonus
	if strings.HasPrefix(b, a) {
		return 0.9
	}
	// Calculate Levenshtein distance
	distance := levenshteinDistance(a, b)
	maxLen := float64(max(len(a), len(b)))

	// Convert distance to similarity score (0 to 1)
	similarity := 1.0 - float64(distance)/maxLen

	return similarity
}

func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1, // deletion
				min(matrix[i][j-1]+1, // insertion
					matrix[i-1][j-1]+cost)) // substitution
		}
	}

	return matrix[len(a)][len(b)]
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	var (
		lines         []string
		currentLine   []string
		currentLength int
	)
	for _, word := range words {
		if currentLength+len(word)+1 > width {
			if len(currentLine) > 0 {
				lines = append(lines, strings.Join(currentLine, " "))
				currentLine = []string{word}
				currentLength = len(word)
			} else {
				lines = append(lines, word)
			}
		} else {
			currentLine = append(currentLine, word)
			if currentLength == 0 {
				currentLength = len(word)
			} else {
				currentLength += len(word) + 1
			}
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}
	return lines
}
