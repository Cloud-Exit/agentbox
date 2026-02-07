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

// Package statusbar renders a persistent top-line status bar using ANSI
// scroll regions. The bar stays fixed at row 1 while the container's
// terminal output scrolls within rows 2..height.
package statusbar

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI sequences
const (
	saveCursor    = "\033[s"
	restoreCursor = "\033[r" // reset scroll region
	clearLine     = "\033[K"
	resetAttrs    = "\033[0m"
	bgDarkGrey    = "\033[48;5;236m"
	fgWhite       = "\033[97m"
)

var active bool

// Show renders the status bar at line 1 and sets the scroll region to
// rows 2..height so the container output doesn't overwrite it.
// No-op if stdout is not a TTY.
func Show(version, agent string) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return
	}

	width, height, err := term.GetSize(fd)
	if err != nil || height < 3 || width < 10 {
		return
	}

	text := fmt.Sprintf(" ExitBox %s - %s ", version, agent)
	if len(text) < width {
		text += strings.Repeat(" ", width-len(text))
	}

	// Move to row 1, render bar
	fmt.Fprintf(os.Stdout, "\033[1;1H%s%s%s%s%s", bgDarkGrey, fgWhite, text, resetAttrs, clearLine)

	// Set scroll region to rows 2..height
	fmt.Fprintf(os.Stdout, "\033[2;%dr", height)

	// Move cursor to row 2
	fmt.Fprintf(os.Stdout, "\033[2;1H")

	active = true
}

// Hide resets the scroll region and clears the status bar line.
// No-op if Show was never called.
func Hide() {
	if !active {
		return
	}
	active = false

	// Reset scroll region
	fmt.Fprint(os.Stdout, restoreCursor)

	// Move to row 1, clear the bar
	fmt.Fprintf(os.Stdout, "\033[1;1H%s", clearLine)

	// Move cursor below where the content was
	fmt.Fprint(os.Stdout, "\033[2;1H")
}
