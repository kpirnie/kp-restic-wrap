/**
 * KP - Restic Backup Wrapper
 *
 * Managed FUSE mount lifecycle: spawns `restic mount` as a child
 * process, waits for the mountpoint to become live, and guarantees
 * unmount on shutdown with a fusermount fallback.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package restic

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Mount starts `restic mount` at mountpoint, streaming restic's
// output. The returned wait blocks until the child exits; cleanup
// interrupts the child and force-unmounts if needed. Callers must
// invoke cleanup on shutdown paths.
func (c *Client) Mount(mountpoint string) (wait func() error, cleanup func(), err error) {

	cmd := c.command("mount", mountpoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start the child
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("starting restic mount: %w", err)
	}

	// wait blocks until restic exits (unmount, signal, or error)
	wait = func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("restic mount: %w", err)
		}
		return nil
	}

	// cleanup interrupts restic, giving it a moment to unmount
	// cleanly, then falls back to fusermount -u if the mountpoint is
	// still live: a stale FUSE mount blocks every later access with
	// "transport endpoint is not connected"
	cleanup = func() {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
		}
		time.Sleep(2 * time.Second)
		if IsMounted(mountpoint) {
			exec.Command("fusermount", "-u", mountpoint).Run()
		}
	}

	return wait, cleanup, nil
}

// WaitForMount polls until the mountpoint is live or the timeout
// elapses, so callers can announce readiness accurately.
func WaitForMount(mountpoint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsMounted(mountpoint) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// IsMounted reports whether path is currently a mountpoint, by
// comparing its device ID against its parent directory's: a
// mountpoint sits on a different device than its parent.
func IsMounted(path string) bool {

	var st, pst syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return false
	}
	if err := syscall.Stat(path+"/..", &pst); err != nil {
		return false
	}

	return st.Dev != pst.Dev
}
