/**
 * KP - Restic Backup Wrapper
 *
 * Interactive mount: select a backup name, input a mountpoint, and
 * hold the FUSE mount in the foreground until interrupted.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package mount

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kpirnie/kp-restic-wrap/internal/config"
	"github.com/kpirnie/kp-restic-wrap/internal/logger"
	"github.com/kpirnie/kp-restic-wrap/internal/restic"
)

// Run walks the interactive mount flow and blocks until the mount
// is released.
func Run(cfg *config.Config) error {

	in := bufio.NewReader(os.Stdin)

	// step 1: pick the backup name from the configured entries
	fmt.Println("Backups:")
	for i, b := range cfg.Backups {
		fmt.Printf("  %d) %s\n", i+1, b.Name)
	}
	fmt.Println()
	var entry config.Backup
	for {
		fmt.Print("Mount which backup? [1]: ")
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(cfg.Backups) {
			fmt.Println("  no such backup")
			continue
		}
		entry = cfg.Backups[n-1]
		break
	}

	// step 2: mountpoint input
	var mountpoint string
	for {
		fmt.Print("Mount at path: ")
		line, _ := in.ReadString('\n')
		mountpoint = strings.TrimSpace(line)
		if mountpoint == "" {
			fmt.Println("  a mountpoint is required")
			continue
		}

		// create if missing
		if err := os.MkdirAll(mountpoint, 0o700); err != nil {
			fmt.Printf("  cannot create %s: %v\n", mountpoint, err)
			continue
		}

		// refuse a non-empty directory: mounting over content hides it
		entries, err := os.ReadDir(mountpoint)
		if err != nil {
			fmt.Printf("  cannot read %s: %v\n", mountpoint, err)
			continue
		}
		if len(entries) > 0 {
			fmt.Printf("  %s is not empty\n", mountpoint)
			continue
		}

		// refuse an existing mount
		if restic.IsMounted(mountpoint) {
			fmt.Printf("  %s is already a mountpoint\n", mountpoint)
			continue
		}

		break
	}

	// start the managed mount
	client := restic.New(cfg, entry.Name)
	logger.Info("mounting %s at %s", entry.Name, mountpoint)
	wait, cleanup, err := client.Mount(mountpoint)
	if err != nil {
		return err
	}

	// forward interrupt/terminate into a clean unmount
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nUnmounting...")
		cleanup()
	}()

	// announce readiness once the kernel reports the mount live
	if restic.WaitForMount(mountpoint, 30*time.Second) {
		logger.Info("mounted - browse %s, Ctrl-C to unmount", mountpoint)
	} else {
		logger.Warn("mount not confirmed after 30s - it may still be connecting")
	}

	// block until restic exits, then ensure the mountpoint is clear
	werr := wait()
	cleanup()
	if werr != nil {
		return werr
	}

	logger.Info("unmounted")
	return nil
}
