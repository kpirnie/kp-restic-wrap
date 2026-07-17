/**
 * KP - Restic Backup Wrapper
 *
 * A CLI wrapper around restic providing configuration-driven
 * backup, restore, and mount operations against S3-compatible storage.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kpirnie/kp-restic-wrap/internal/logger"
)

// usage prints the top-level command usage to stderr.
func usage() {
	fmt.Fprintf(os.Stderr, `Usage: kp <command> [options]

Commands:
  configure   Create or edit the configuration file
  backup      Run a backup, then apply retention and prune
  restore     Interactively restore from a snapshot
  mount       Mount the repository via FUSE

Options:
  --config    Path to config file (default /etc/kp/config.yaml)
`)
}

// main dispatches the requested subcommand.
func main() {

	// no subcommand given
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// pull the subcommand and shift args for flag parsing
	cmd := os.Args[1]
	args := os.Args[2:]

	// shared flagset: every subcommand accepts --config
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	cfgPath := fs.String("config", "/etc/kp/config.yaml", "path to config file")
	fs.Parse(args)

	// dispatch
	var err error
	switch cmd {
	case "configure":
		err = runConfigure(*cfgPath, fs.Args())
	case "backup":
		err = runBackup(*cfgPath, fs.Args())
	case "restore":
		err = runRestore(*cfgPath, fs.Args())
	case "mount":
		err = runMount(*cfgPath, fs.Args())
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "kp: unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}

	// report failure and exit non-zero
	if err != nil {
		logger.Error("kp: %v", err)
		os.Exit(1)
	}
}
