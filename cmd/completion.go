// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell autocompletion",
	Long: `Generate autocompletion for your shell.

If no shell is specified, the current shell is detected automatically.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgs:         []string{"bash", "zsh", "fish"},
	DisableFlagParsing: false,
	Run: func(cmd *cobra.Command, args []string) {
		shell := ""
		if len(args) > 0 {
			shell = args[0]
		} else {
			shell = detectShell()
		}

		switch shell {
		case "bash":
			generateBash()
		case "zsh":
			generateZsh()
		case "fish":
			generateFish()
		default:
			ui.Errorf("Unsupported shell: %s (supported: bash, zsh, fish)", shell)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// detectShell returns the name of the user's current shell.
func detectShell() string {
	// Check $SHELL (login shell)
	sh := os.Getenv("SHELL")
	if sh != "" {
		base := filepath.Base(sh)
		switch base {
		case "bash", "zsh", "fish":
			return base
		}
	}

	// Check parent process name via /proc on Linux
	ppidComm := fmt.Sprintf("/proc/%d/comm", os.Getppid())
	if data, err := os.ReadFile(ppidComm); err == nil {
		name := strings.TrimSpace(string(data))
		switch name {
		case "bash", "zsh", "fish":
			return name
		}
	}

	// Default to bash
	return "bash"
}

// showHints prints usage hints to stderr, but only when stdout is a terminal
// (i.e., not being piped to eval or redirected to a file).
func showHints(lines ...string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func generateBash() {
	if err := rootCmd.GenBashCompletionV2(os.Stdout, true); err != nil {
		ui.Errorf("Failed to generate bash completion: %v", err)
		os.Exit(1)
	}

	showHints(
		"",
		"# To enable autocompletion, add this to your ~/.bashrc:",
		"#",
		"#   eval \"$(exitbox completion bash)\"",
		"#",
		"# Or generate a file and source it:",
		"#",
		"#   exitbox completion bash > ~/.local/share/bash-completion/completions/exitbox",
	)
}

func generateZsh() {
	if err := rootCmd.GenZshCompletion(os.Stdout); err != nil {
		ui.Errorf("Failed to generate zsh completion: %v", err)
		os.Exit(1)
	}

	showHints(
		"",
		"# To enable autocompletion, add this to your ~/.zshrc:",
		"#",
		"#   eval \"$(exitbox completion zsh)\"",
		"#",
		"# Or generate a file (for faster shell startup):",
		"#",
		"#   exitbox completion zsh > ~/.zfunc/_exitbox",
		"#   # then add to ~/.zshrc (before compinit): fpath=(~/.zfunc $fpath)",
	)
}

func generateFish() {
	if err := rootCmd.GenFishCompletion(os.Stdout, true); err != nil {
		ui.Errorf("Failed to generate fish completion: %v", err)
		os.Exit(1)
	}

	showHints(
		"",
		"# To enable autocompletion, run:",
		"#",
		"#   exitbox completion fish > ~/.config/fish/completions/exitbox.fish",
		"#",
		"# Fish will pick it up automatically on next shell start.",
	)
}
