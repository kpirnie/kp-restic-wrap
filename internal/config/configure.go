/**
 * KP - Restic Backup Wrapper
 *
 * Interactive configuration editor: walks the shared settings,
 * manages the backup entry list, handles no-echo password entry,
 * and persists the result with restrictive permissions.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Configure runs the interactive walk. It returns the resulting
// configuration, whether it was freshly created (no prior file), and
// the previous password when the user changed it on an existing
// config, so the caller can rotate the repository keys.
func Configure(path string) (cfg *Config, created bool, oldPassword string, err error) {

	// load existing config, or start from defaults on first run
	cfg, lerr := Load(path)
	if lerr != nil {
		cfg = Defaults()
		created = true
		fmt.Printf("Creating new configuration at %s\n\n", path)
	} else {
		fmt.Printf("Editing configuration at %s (Enter keeps the current value)\n\n", path)
	}

	in := bufio.NewReader(os.Stdin)

	// s3 settings
	cfg.S3.Endpoint = promptString(in, "S3 endpoint", cfg.S3.Endpoint)
	cfg.S3.Key = promptString(in, "S3 key", cfg.S3.Key)
	cfg.S3.Secret = promptString(in, "S3 secret", cfg.S3.Secret)
	cfg.S3.Bucket = promptString(in, "S3 bucket", cfg.S3.Bucket)
	cfg.S3.Region = promptString(in, "S3 region", cfg.S3.Region)

	// temp path
	cfg.TempPath = promptString(in, "Temp path", cfg.TempPath)

	// encryption password
	oldPassword = cfg.Encryption.Password
	newPass, changed, perr := promptPassword(in, cfg.Encryption.Password != "")
	if perr != nil {
		return nil, false, "", perr
	}
	if changed {
		cfg.Encryption.Password = newPass
	} else {
		// unchanged: signal no rotation needed
		oldPassword = ""
	}
	if created {
		// fresh config never needs rotation
		oldPassword = ""
	}

	// backup entries
	cfg.Backups = promptBackups(in, cfg.Backups)

	// validate before persisting
	if err := cfg.Validate(); err != nil {
		return nil, false, "", fmt.Errorf("configuration invalid: %w", err)
	}

	// persist
	if err := cfg.Save(path); err != nil {
		return nil, false, "", err
	}
	fmt.Printf("\nConfiguration saved to %s\n", path)

	return cfg, created, oldPassword, nil
}

// promptBackups manages the backup entry list: shows the current
// entries, then accepts add/edit/remove commands until done.
func promptBackups(in *bufio.Reader, current []Backup) []Backup {

	backups := append([]Backup(nil), current...)

	for {
		// show current state
		fmt.Println("\nBackup entries:")
		if len(backups) == 0 {
			fmt.Println("  (none)")
		}
		for i, b := range backups {
			fmt.Printf("  %d) %s  %s  keep %dd  %d exclude(s)\n",
				i+1, b.Name, b.StartPath, b.RetentionDays, len(b.Excludes))
		}
		fmt.Print("Add (a), edit by number (e.g. 2), remove by number (e.g. -2), or Enter when done: ")

		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)

		// done
		if line == "" {
			fmt.Println()
			return backups
		}

		// addition
		if strings.EqualFold(line, "a") {
			backups = append(backups, promptBackupEntry(in, DefaultBackup()))
			continue
		}

		// removal: -N
		if strings.HasPrefix(line, "-") {
			n, err := strconv.Atoi(line[1:])
			if err != nil || n < 1 || n > len(backups) {
				fmt.Println("  no such entry")
				continue
			}
			backups = append(backups[:n-1], backups[n:]...)
			continue
		}

		// edit: N
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(backups) {
			fmt.Println("  no such entry")
			continue
		}
		backups[n-1] = promptBackupEntry(in, backups[n-1])
	}
}

// promptBackupEntry walks the fields of a single backup entry with
// the given values as defaults.
func promptBackupEntry(in *bufio.Reader, b Backup) Backup {
	fmt.Println()
	b.Name = promptString(in, "Backup name", b.Name)
	b.StartPath = promptString(in, "Start path", b.StartPath)
	b.RetentionDays = promptInt(in, "Retention days", b.RetentionDays)
	b.Excludes = promptExcludes(in, b.Excludes)
	return b
}

// promptString asks for a string value, returning def when the user
// presses Enter without input.
func promptString(in *bufio.Reader, label string, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// promptInt asks for an integer value, re-prompting on bad input and
// returning def when the user presses Enter without input.
func promptInt(in *bufio.Reader, label string, def int) int {
	for {
		fmt.Printf("%s [%d]: ", label, def)
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		v, err := strconv.Atoi(line)
		if err != nil || v < 1 {
			fmt.Println("  enter a whole number of 1 or more")
			continue
		}
		return v
	}
}

// promptExcludes manages the exclude pattern list: shows the current
// entries, then accepts add/remove/done commands.
func promptExcludes(in *bufio.Reader, current []string) []string {

	excludes := append([]string(nil), current...)

	for {
		// show current state
		fmt.Println("\nExclude patterns:")
		if len(excludes) == 0 {
			fmt.Println("  (none)")
		}
		for i, e := range excludes {
			fmt.Printf("  %d) %s\n", i+1, e)
		}
		fmt.Print("Add pattern, remove by number (e.g. -2), or Enter when done: ")

		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)

		// done
		if line == "" {
			return excludes
		}

		// removal: -N
		if strings.HasPrefix(line, "-") {
			n, err := strconv.Atoi(line[1:])
			if err != nil || n < 1 || n > len(excludes) {
				fmt.Println("  no such entry")
				continue
			}
			excludes = append(excludes[:n-1], excludes[n:]...)
			continue
		}

		// addition
		excludes = append(excludes, line)
	}
}

// promptPassword handles no-echo password entry. When a password
// already exists it offers to keep it; a new password must be entered
// twice. Returns the password, whether it changed, and any error.
func promptPassword(in *bufio.Reader, hasExisting bool) (string, bool, error) {

	// offer to keep the current password
	if hasExisting {
		fmt.Print("Change encryption password? [y/N]: ")
		line, _ := in.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			return "", false, nil
		}
		// changing the password on existing repositories requires a
		// key rotation on every repo, otherwise the config no longer
		// opens them; the caller handles that using the returned old
		// password
		fmt.Println("NOTE: every repository key will be rotated to match.")
	}

	// entered twice, no echo
	for {
		fmt.Print("Encryption password: ")
		p1, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", false, fmt.Errorf("reading password: %w", err)
		}
		if len(p1) == 0 {
			fmt.Println("  password cannot be empty")
			continue
		}
		fmt.Print("Confirm password: ")
		p2, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", false, fmt.Errorf("reading password: %w", err)
		}
		if string(p1) != string(p2) {
			fmt.Println("  passwords do not match, try again")
			continue
		}
		return string(p1), true, nil
	}
}
