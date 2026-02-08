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

package wizard

import "github.com/cloud-exit/exitbox/internal/config"

// domainCategory represents one section of the domain allowlist.
type domainCategory struct {
	Name    string   // display name (e.g. "AI Providers")
	Key     string   // yaml field key (e.g. "ai_providers")
	Domains []string // the domains in this category
}

// allowlistToCategories converts a config.Allowlist into the 5 editable categories.
func allowlistToCategories(al *config.Allowlist) []domainCategory {
	return []domainCategory{
		{Name: "AI Providers", Key: "ai_providers", Domains: copyStrings(al.AIProviders)},
		{Name: "Development", Key: "development", Domains: copyStrings(al.Development)},
		{Name: "Cloud Services", Key: "cloud_services", Domains: copyStrings(al.CloudServices)},
		{Name: "Common Services", Key: "common_services", Domains: copyStrings(al.CommonServices)},
		{Name: "Custom", Key: "custom", Domains: copyStrings(al.Custom)},
	}
}

// categoriesToAllowlist converts editable categories back to a config.Allowlist.
func categoriesToAllowlist(cats []domainCategory) *config.Allowlist {
	al := &config.Allowlist{Version: 1}
	for _, c := range cats {
		switch c.Key {
		case "ai_providers":
			al.AIProviders = copyStrings(c.Domains)
		case "development":
			al.Development = copyStrings(c.Domains)
		case "cloud_services":
			al.CloudServices = copyStrings(c.Domains)
		case "common_services":
			al.CommonServices = copyStrings(c.Domains)
		case "custom":
			al.Custom = copyStrings(c.Domains)
		}
	}
	return al
}

// countDomains returns the total number of domains across all categories.
func countDomains(cats []domainCategory) int {
	n := 0
	for _, c := range cats {
		n += len(c.Domains)
	}
	return n
}

// countCustomDomains returns the number of custom domains.
func countCustomDomains(cats []domainCategory) int {
	for _, c := range cats {
		if c.Key == "custom" {
			return len(c.Domains)
		}
	}
	return 0
}

func copyStrings(ss []string) []string {
	if ss == nil {
		return nil
	}
	out := make([]string, len(ss))
	copy(out, ss)
	return out
}
