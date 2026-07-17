/**
 * KP - Restic Backup Wrapper
 *
 * Subcommand dispatch stubs, replaced as command packages are implemented.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kpirnie/kp-restic-wrap/internal/backup"
	"github.com/kpirnie/kp-restic-wrap/internal/config"
	"github.com/kpirnie/kp-restic-wrap/internal/logger"
	"github.com/kpirnie/kp-restic-wrap/internal/mount"
	"github.com/kpirnie/kp-restic-wrap/internal/restic"
	"github.com/kpirnie/kp-restic-wrap/internal/restore"
)

// runConfigure launches the interactive configuration editor, rotates
// every repository key on password change, and offers to initialize
// any repository that does not yet exist.
func runConfigure(cfgPath string, args []string) error {

	// verify restic is available before anything else
	if _, err := restic.Version(); err != nil {
		return err
	}

	// interactive walk
	cfg, _, oldPassword, err := config.Configure(cfgPath)
	if err != nil {
		return err
	}

	// password changed on an existing config: rotate every existing
	// repository key using the old password, since the saved config
	// now holds the new one and would otherwise be locked out
	if oldPassword != "" {
		rotator := *cfg
		rotator.Encryption.Password = oldPassword
		for _, b := range cfg.Backups {
			client := restic.New(&rotator, b.Name)
			exists, err := client.RepoExists()
			if err != nil {
				logger.Error("checking %s: %v", b.Name, err)
				continue
			}
			if !exists {
				continue
			}
			logger.Info("rotating repository key for %s", b.Name)
			if err := client.ChangePassword(cfg.Encryption.Password); err != nil {
				logger.Error("key rotation failed for %s - repository still uses the old password: %v", b.Name, err)
				continue
			}
			logger.Info("repository key rotated for %s", b.Name)
		}
	}

	// offer to initialize any repository that does not exist
	in := bufio.NewReader(os.Stdin)
	for _, b := range cfg.Backups {
		client := restic.New(cfg, b.Name)
		exists, err := client.RepoExists()
		if err != nil {
			logger.Warn("could not check repository %s: %v", b.Name, err)
			continue
		}
		if !exists {
			fmt.Printf("Repository for %q does not exist. Initialize it now? [Y/n]: ", b.Name)
			line, _ := in.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "" || strings.EqualFold(line, "y") {
				if err := client.Init(); err != nil {
					logger.Error("initializing %s: %v", b.Name, err)
				}
			}
		}
	}

	return nil
}

// runBackup executes the backup, retention, and prune cycle.
func runBackup(cfgPath string, args []string) error {

	// verify restic is available
	if _, err := restic.Version(); err != nil {
		return err
	}

	// load and validate config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	return backup.Run(cfg)
}

// runRestore starts the interactive restore flow.
func runRestore(cfgPath string, args []string) error {

	// verify restic is available
	if _, err := restic.Version(); err != nil {
		return err
	}

	// load and validate config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	return restore.Run(cfg)
}

// runMount walks the interactive mount flow.
func runMount(cfgPath string, args []string) error {

	// verify restic is available
	if _, err := restic.Version(); err != nil {
		return err
	}

	// load and validate config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	return mount.Run(cfg)
}
