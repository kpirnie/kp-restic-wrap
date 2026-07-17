/**
 * KP - Restic Backup Wrapper
 *
 * Subcommand dispatch stubs, replaced as command packages are implemented.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package main

import "fmt"

// runConfigure launches the interactive configuration editor.
func runConfigure(cfgPath string, args []string) error {
	return fmt.Errorf("configure: not yet implemented")
}

// runBackup executes the backup, retention, and prune cycle.
func runBackup(cfgPath string, args []string) error {
	return fmt.Errorf("backup: not yet implemented")
}

// runRestore starts the interactive restore flow.
func runRestore(cfgPath string, args []string) error {
	return fmt.Errorf("restore: not yet implemented")
}

// runMount mounts the repository at the requested mountpoint.
func runMount(cfgPath string, args []string) error {
	return fmt.Errorf("mount: not yet implemented")
}
