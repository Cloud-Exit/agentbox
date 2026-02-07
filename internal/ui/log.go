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

package ui

import (
	"fmt"
	"os"
)

// Verbose controls whether debug messages are printed.
var Verbose bool

// Info prints an informational message to stdout.
func Info(msg string) {
	fmt.Printf("%s[INFO]%s %s\n", Cyan, NC, msg)
}

// Infof prints a formatted informational message to stdout.
func Infof(format string, a ...any) {
	Info(fmt.Sprintf(format, a...))
}

// Success prints a success message to stdout.
func Success(msg string) {
	fmt.Printf("%s[OK]%s %s\n", Green, NC, msg)
}

// Successf prints a formatted success message to stdout.
func Successf(format string, a ...any) {
	Success(fmt.Sprintf(format, a...))
}

// Warn prints a warning message to stderr.
func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s[WARN]%s %s\n", Yellow, NC, msg)
}

// Warnf prints a formatted warning message to stderr.
func Warnf(format string, a ...any) {
	Warn(fmt.Sprintf(format, a...))
}

// Error prints an error message to stderr and exits with code 1.
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", Red, NC, msg)
	os.Exit(1)
}

// Errorf prints a formatted error message to stderr and exits with code 1.
func Errorf(format string, a ...any) {
	Error(fmt.Sprintf(format, a...))
}

// ErrorNoExit prints an error message to stderr without exiting.
func ErrorNoExit(msg string) {
	fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", Red, NC, msg)
}

// Debug prints a debug message to stderr (only when Verbose is true).
func Debug(msg string) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "%s[DEBUG]%s %s\n", Dim, NC, msg)
	}
}

// Debugf prints a formatted debug message to stderr.
func Debugf(format string, a ...any) {
	Debug(fmt.Sprintf(format, a...))
}

// Cecho prints colored text to stdout.
func Cecho(msg, color string) {
	fmt.Printf("%s%s%s\n", color, msg, NC)
}
