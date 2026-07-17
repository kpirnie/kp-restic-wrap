/**
 * KP - Restic Backup Wrapper
 *
 * Backup orchestration: runs every configured backup entry through
 * a bounded worker pool, applying retention and pruning per entry.
 * Output is captured per entry and emitted on completion so
 * concurrent runs do not interleave.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package backup

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/kpirnie/kp-restic-wrap/internal/config"
	"github.com/kpirnie/kp-restic-wrap/internal/logger"
	"github.com/kpirnie/kp-restic-wrap/internal/restic"
)

// result carries the outcome of one backup entry through the pool.
type result struct {
	name    string
	output  []byte
	warned  bool
	failed  bool
	failErr error
}

// Run executes the full backup cycle for every configured entry via
// a worker pool capped at config.Concurrency. It returns an error if
// any entry fully failed; partial-read warnings (restic exit 3) are
// logged but do not count as failures.
func Run(cfg *config.Config) error {

	workers := config.Concurrency()
	logger.Info("running %d backup(s), %d at a time", len(cfg.Backups), workers)

	// bounded pool: jobs in, results out
	jobs := make(chan config.Backup)
	results := make(chan result)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range jobs {
				results <- runOne(cfg, b)
			}
		}()
	}

	// feed the pool, close results when all workers finish
	go func() {
		for _, b := range cfg.Backups {
			jobs <- b
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	// collect: emit each entry's captured output as it completes
	var failures []string
	for r := range results {

		// prefix each captured line with the entry name
		fmt.Printf("\n===== %s =====\n", r.name)
		for _, line := range strings.Split(strings.TrimRight(string(r.output), "\n"), "\n") {
			fmt.Printf("[%s] %s\n", r.name, line)
		}

		switch {
		case r.failed:
			logger.Error("%s: backup failed: %v", r.name, r.failErr)
			failures = append(failures, r.name)
		case r.warned:
			logger.Warn("%s: completed with unreadable files (snapshot was still created)", r.name)
		default:
			logger.Info("%s: backup complete", r.name)
		}
	}

	// non-zero exit only for full failures, keeping cron alerts honest
	if len(failures) > 0 {
		return fmt.Errorf("backup(s) failed: %s", strings.Join(failures, ", "))
	}

	return nil
}

// runOne executes backup, retention, and prune for a single entry
// with all output captured.
func runOne(cfg *config.Config, b config.Backup) result {

	r := result{name: b.Name}
	client := restic.New(cfg, b.Name)

	// the start path must exist before handing it to restic
	if _, err := os.Stat(b.StartPath); err != nil {
		r.failed = true
		r.failErr = fmt.Errorf("start path %s: %w", b.StartPath, err)
		return r
	}

	// verify the repository is reachable and initialized
	exists, err := client.RepoExists()
	if err != nil {
		r.failed = true
		r.failErr = err
		return r
	}
	if !exists {
		r.failed = true
		r.failErr = fmt.Errorf("repository %s is not initialized - run `kp configure`", client.RepoURL())
		return r
	}

	// backup: one snapshot of the start path with configured excludes
	args := []string{"backup", b.StartPath}
	for _, e := range b.Excludes {
		args = append(args, "--exclude", e)
	}
	out, code, err := client.RunCaptured(args...)
	r.output = out
	switch {
	case err == nil:
		// clean run
	case code == 3:
		// restic exit 3: snapshot created but some files could not be
		// read (vanished files, permission denials) - a warning, not
		// a failure
		r.warned = true
	default:
		r.failed = true
		r.failErr = err
		return r
	}

	// retention: drop snapshots older than the configured window and
	// prune unreferenced data in the same pass
	keep := fmt.Sprintf("%dd", b.RetentionDays)
	out, _, err = client.RunCaptured("forget", "--keep-within", keep, "--prune")
	r.output = append(r.output, out...)
	if err != nil {
		r.failed = true
		r.failErr = fmt.Errorf("retention: %w", err)
	}

	return r
}
