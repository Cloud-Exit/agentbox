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

import "fmt"

// LogoSmall prints the ExitBox ASCII logo.
func LogoSmall() {
	fmt.Print(Cyan)
	fmt.Println(`  _____      _ _   ____            `)
	fmt.Println(` | ____|_  _(_) |_| __ )  _____  __`)
	fmt.Println(` |  _| \ \/ / | __|  _ \ / _ \ \/ /`)
	fmt.Println(` | |___ >  <| | |_| |_) | (_) >  < `)
	fmt.Println(` |_____/_/\_\_|\__|____/ \___/_/\_\`)
	fmt.Print(NC)
	fmt.Printf("%s         by Cloud Exit (https://cloud-exit.com)%s\n", Dim, NC)
}

// Logo prints the full ExitBox logo with tagline.
func Logo() {
	LogoSmall()
	fmt.Println()
	fmt.Printf("%sMulti-Agent Container Sandbox%s\n", Dim, NC)
}
