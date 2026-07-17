/**
 * KP - Restic Backup Wrapper
 *
 * Interactive restore: select a backup name, select a snapshot by
 * date/time, optionally limit to specific paths, and restore into
 * an input target path.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package restore

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kpirnie/kp-restic-wrap/internal/config"
	"github.com/kpirnie/kp-restic-wrap/internal/logger"
	"github.com/kpirnie/kp-restic-wrap/internal/restic"
)

// Run walks the interactive restore flow.
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
		fmt.Print("Restore from which backup? [1]: ")
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

	client := restic.New(cfg, entry.Name)

	// step 2: pick the snapshot by date/time, newest first
	snaps, err := client.Snapshots()
	if err != nil {
		return err
	}
	if len(snaps) == 0 {
		return fmt.Errorf("no snapshots in repository %s", client.RepoURL())
	}

	fmt.Printf("\nSnapshots for %s:\n\n", entry.Name)
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		fmt.Printf("  %d) %s  %s  %s\n",
			len(snaps)-i,
			s.ShortID,
			s.Time.Local().Format("2006-01-02 15:04:05"),
			strings.Join(s.Paths, ", "))
	}
	fmt.Println()

	var chosen restic.Snapshot
	for {
		fmt.Print("Restore which snapshot? [1]: ")
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(snaps) {
			fmt.Println("  no such snapshot")
			continue
		}
		chosen = snaps[len(snaps)-n]
		break
	}

	// step 3: target path input
	var target string
	for {
		fmt.Print("Restore to path: ")
		line, _ := in.ReadString('\n')
		target = strings.TrimSpace(line)
		if target == "" {
			fmt.Println("  a target path is required")
			continue
		}
		break
	}

	// scope: full snapshot or specific paths within it
	var includes []string
	fmt.Print("Restore the full snapshot, or a specific path within it? [full]: ")
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" && !strings.EqualFold(line, "full") {
		includes = append(includes, line)
		// allow additional paths until Enter
		for {
			fmt.Print("Another path (Enter when done): ")
			more, _ := in.ReadString('\n')
			more = strings.TrimSpace(more)
			if more == "" {
				break
			}
			includes = append(includes, more)
		}
	}

	// restoring over the original location overwrites live data, so
	// require explicit confirmation before proceeding
	fmt.Printf("\nRestore %s snapshot %s into %s\n", entry.Name, chosen.ShortID, target)
	if len(includes) > 0 {
		fmt.Printf("Limited to: %s\n", strings.Join(includes, ", "))
	}
	fmt.Print("Proceed? [y/N]: ")
	line, _ = in.ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") {
		fmt.Println("Restore cancelled.")
		return nil
	}

	// restore with live progress
	logger.Info("restoring %s snapshot %s to %s", entry.Name, chosen.ShortID, target)
	args := []string{"restore", chosen.ID, "--target", target}
	for _, inc := range includes {
		args = append(args, "--include", inc)
	}
	if err := client.Run(args...); err != nil {
		return err
	}

	logger.Info("restore complete")
	return nil
}
