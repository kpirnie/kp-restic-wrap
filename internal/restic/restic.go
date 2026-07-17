/**
 * KP - Restic Backup Wrapper
 *
 * Restic process execution: environment construction, repository
 * URL handling, and command invocation. This is the only package
 * that shells out to the restic binary.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package restic

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/kpirnie/kp-restic-wrap/internal/config"
)

// Client executes restic commands against a single named repository.
type Client struct {
	cfg  *config.Config
	name string
}

// New returns a Client bound to the named backup's repository within
// the given configuration.
func New(cfg *config.Config, name string) *Client {
	return &Client{cfg: cfg, name: name}
}

// RepoURL returns the repository URL this client is bound to.
func (c *Client) RepoURL() string {
	return c.cfg.RepoURL(c.name)
}

// env builds the child process environment. Credentials and the
// repository password are passed via environment only, never as
// command-line arguments, so they are not visible in process lists.
func (c *Client) env() []string {
	return append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+c.cfg.S3.Key,
		"AWS_SECRET_ACCESS_KEY="+c.cfg.S3.Secret,
		"AWS_DEFAULT_REGION="+c.cfg.S3.Region,
		"RESTIC_REPOSITORY="+c.RepoURL(),
		"RESTIC_PASSWORD="+c.cfg.Encryption.Password,
		// point restic staging at real disk: tmpfs-backed /tmp has
		// caused corrupted restores when backups exceed available RAM
		"TMPDIR="+c.cfg.TempPath,
	)
}

// command builds an exec.Cmd for the given restic arguments with the
// repository environment applied.
func (c *Client) command(args ...string) *exec.Cmd {
	cmd := exec.Command("restic", args...)
	cmd.Env = c.env()
	return cmd
}

// Run executes a restic command, streaming its output directly to
// the terminal. Used for interactive and progress-producing commands.
func (c *Client) Run(args ...string) error {
	cmd := c.command(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic %s: %w", args[0], err)
	}
	return nil
}

// RunCaptured executes a restic command with all output captured,
// for concurrent runs where interleaved terminal output is unwanted.
// It returns the combined output and the command's exit code.
func (c *Client) RunCaptured(args ...string) (output []byte, exitCode int, err error) {
	cmd := c.command(args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	rerr := cmd.Run()
	if rerr == nil {
		return buf.Bytes(), 0, nil
	}
	if ee, ok := rerr.(*exec.ExitError); ok {
		return buf.Bytes(), ee.ExitCode(), fmt.Errorf("restic %s: %w", args[0], rerr)
	}
	return buf.Bytes(), -1, fmt.Errorf("restic %s: %w", args[0], rerr)
}

// Output executes a restic command and returns its captured stdout.
// Stderr is captured separately and included in any error.
func (c *Client) Output(args ...string) ([]byte, error) {
	cmd := c.command(args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("restic %s: %w: %s", args[0], err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// RepoExists reports whether the bound repository is initialized.
// Restic exits with code 10 when the repository does not exist; any
// other failure (auth, network) is returned as an error.
func (c *Client) RepoExists() (bool, error) {
	cmd := c.command("cat", "config")
	var stderr bytes.Buffer
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 10 {
		return false, nil
	}
	return false, fmt.Errorf("checking repository: %w: %s", err, stderr.String())
}

// Init initializes the bound repository.
func (c *Client) Init() error {
	return c.Run("init")
}

// ChangePassword rotates the repository password from the currently
// configured one to newPassword. The new password is handed to restic
// via a short-lived owner-only file, since restic offers no
// environment variable for the new key.
func (c *Client) ChangePassword(newPassword string) error {

	// stage the new password in the configured temp path
	f, err := os.CreateTemp(c.cfg.TempPath, ".kp-newpass-*")
	if err != nil {
		return fmt.Errorf("staging new password: %w", err)
	}
	defer os.Remove(f.Name())

	// owner-only before content is written
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return fmt.Errorf("securing password file: %w", err)
	}
	if _, err := f.WriteString(newPassword); err != nil {
		f.Close()
		return fmt.Errorf("writing password file: %w", err)
	}
	f.Close()

	// current password comes from env; new one from the staged file
	return c.Run("key", "passwd", "--new-password-file", f.Name())
}

// Version returns restic's version string, verifying the binary is
// present and executable.
func Version() (string, error) {
	out, err := exec.Command("restic", "version").Output()
	if err != nil {
		return "", fmt.Errorf("restic binary not found or not executable: %w", err)
	}
	return string(bytes.TrimSpace(out)), nil
}
