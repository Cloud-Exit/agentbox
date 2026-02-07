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

// Package ui provides terminal output: colors, logging, logos, and welcome screens.
package ui

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes.
var (
	Red     = "\033[0;31m"
	Green   = "\033[0;32m"
	Yellow  = "\033[0;33m"
	Blue    = "\033[0;34m"
	Magenta = "\033[0;35m"
	Cyan    = "\033[0;36m"
	White   = "\033[0;37m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	NC      = "\033[0m" // No Color / Reset
)

func init() {
	if !isTerminal() {
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Magenta = ""
		Cyan = ""
		White = ""
		Bold = ""
		Dim = ""
		NC = ""
	}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
