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
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/cloud-exit/exitbox/internal/config"
	"github.com/cloud-exit/exitbox/internal/kvstore"
	"github.com/cloud-exit/exitbox/internal/ui"
	"github.com/spf13/cobra"
)

func newKVCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kv",
		Short: "Manage the KV store for session persistence",
		Long:  "Low-level key-value store operations for debugging and management.",
	}

	cmd.AddCommand(newKVGetCmd())
	cmd.AddCommand(newKVSetCmd())
	cmd.AddCommand(newKVListCmd())
	cmd.AddCommand(newKVDeleteCmd())
	cmd.AddCommand(newKVBenchCmd())
	return cmd
}

func openKVStore(workspace string) (*kvstore.Store, string) {
	ws := resolveKVWorkspace(workspace)
	dir := config.KVDir(ws)
	store, err := kvstore.Open(kvstore.Options{Dir: dir})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot acquire directory lock") {
			ui.Errorf("KV store is locked by a running session. Stop the session first, or use 'exitbox-kv' from inside the container.")
		}
		ui.Errorf("Failed to open KV store: %v", err)
	}
	return store, ws
}

func resolveKVWorkspace(flag string) string {
	if flag != "" {
		return flag
	}
	return resolveVaultWorkspace(flag)
}

func newKVGetCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a value from the KV store",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			store, _ := openKVStore(workspace)
			defer func() {
				if err := store.Close(); err != nil {
					ui.Warnf("Failed to close KV store: %v", err)
				}
			}()

			val, err := store.Get([]byte(args[0]))
			if err != nil {
				ui.Errorf("%v", err)
			}
			fmt.Println(string(val))
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newKVSetCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a value in the KV store",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			store, ws := openKVStore(workspace)
			defer func() {
				if err := store.Close(); err != nil {
					ui.Warnf("Failed to close KV store: %v", err)
				}
			}()

			if err := store.Set([]byte(args[0]), []byte(args[1])); err != nil {
				ui.Errorf("Failed to set: %v", err)
			}
			ui.Successf("Set '%s' in workspace '%s'", args[0], ws)
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newKVListCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:     "list [prefix]",
		Short:   "List keys in the KV store",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			store, ws := openKVStore(workspace)
			defer func() {
				if err := store.Close(); err != nil {
					ui.Warnf("Failed to close KV store: %v", err)
				}
			}()

			var prefix []byte
			if len(args) > 0 {
				prefix = []byte(args[0])
			}

			count := 0
			err := store.Iterate(prefix, func(key, value []byte) error {
				fmt.Printf("%s = %s\n", key, value)
				count++
				return nil
			})
			if err != nil {
				ui.Errorf("Failed to list: %v", err)
			}
			if count == 0 {
				ui.Infof("No entries found in workspace '%s'", ws)
			}
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newKVDeleteCmd() *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:     "delete <key>",
		Short:   "Delete a key from the KV store",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			store, ws := openKVStore(workspace)
			defer func() {
				if err := store.Close(); err != nil {
					ui.Warnf("Failed to close KV store: %v", err)
				}
			}()

			if err := store.Delete([]byte(args[0])); err != nil {
				ui.Errorf("Failed to delete: %v", err)
			}
			ui.Successf("Deleted '%s' from workspace '%s'", args[0], ws)
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	return cmd
}

func newKVBenchCmd() *cobra.Command {
	var workspace string
	var sizeMB int
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Generate test data to verify KV store expansion",
		Run: func(cmd *cobra.Command, args []string) {
			store, ws := openKVStore(workspace)
			defer func() {
				if err := store.Close(); err != nil {
					ui.Warnf("Failed to close KV store: %v", err)
				}
			}()

			const valSize = 10 * 1024 // 10 KB per key
			numKeys := (sizeMB * 1024 * 1024) / valSize

			ui.Infof("Writing %d keys × 10 KB = %d MB to workspace '%s'...", numKeys, sizeMB, ws)

			value := make([]byte, valSize)
			for i := 0; i < numKeys; i++ {
				if _, err := rand.Read(value); err != nil {
					ui.Errorf("rand.Read: %v", err)
				}
				key := fmt.Sprintf("bench:%04d", i)
				if err := store.Set([]byte(key), value); err != nil {
					ui.Errorf("Failed to set key %d: %v", i, err)
				}
			}

			ui.Successf("Wrote %d MB of test data to workspace '%s'", sizeMB, ws)
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace name")
	cmd.Flags().IntVarP(&sizeMB, "size", "s", 10, "Amount of data to generate in MB")
	return cmd
}

func init() {
	rootCmd.AddCommand(newKVCmd())
}
