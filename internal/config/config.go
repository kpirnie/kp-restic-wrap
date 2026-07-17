/**
 * KP - Restic Backup Wrapper
 *
 * Configuration schema, loading, validation, and persistence.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// S3 holds the S3-compatible storage connection settings shared by
// all configured backups.
type S3 struct {
	Endpoint string `yaml:"endpoint"`
	Key      string `yaml:"key"`
	Secret   string `yaml:"secret"`
	Bucket   string `yaml:"bucket"`
	Region   string `yaml:"region"`
}

// Backup holds a single backup entry: its repository name, source
// path, retention policy, and exclude patterns.
type Backup struct {
	Name          string   `yaml:"name"`
	StartPath     string   `yaml:"start_path"`
	RetentionDays int      `yaml:"retention_days"`
	Excludes      []string `yaml:"excludes"`
}

// Encryption holds the repository encryption settings shared by all
// configured backups.
type Encryption struct {
	Password string `yaml:"password"`
}

// Config is the full application configuration as stored on disk.
type Config struct {
	S3         S3         `yaml:"s3"`
	TempPath   string     `yaml:"temp_path"`
	Encryption Encryption `yaml:"encryption"`
	Backups    []Backup   `yaml:"backups"`
}

// Defaults returns a Config pre-populated with sane default values
// for use as prompts during initial configuration.
func Defaults() *Config {
	return &Config{
		S3: S3{
			Endpoint: "s3.amazonaws.com",
			Region:   "us-east-1",
		},
		TempPath: "/var/tmp/kp",
	}
}

// DefaultBackup returns a Backup entry pre-populated with sane
// defaults for use as prompts when adding a new entry.
func DefaultBackup() Backup {

	// hostname as the default backup name, matching common convention
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "backup"
	}

	return Backup{
		Name:          host,
		StartPath:     "/home",
		RetentionDays: 30,
	}
}

// Load reads and parses the configuration file at path.
// A missing file is returned as a distinct error so callers can
// direct the user to run configure.
func Load(path string) (*Config, error) {

	// read the raw file
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no configuration found at %s - run `kp configure` first", path)
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	// parse the yaml
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// Validate checks the configuration for required values and usable
// paths, returning the first problem found.
func (c *Config) Validate() error {

	// required s3 settings
	if c.S3.Endpoint == "" {
		return errors.New("s3 endpoint is required")
	}
	if c.S3.Key == "" {
		return errors.New("s3 key is required")
	}
	if c.S3.Secret == "" {
		return errors.New("s3 secret is required")
	}
	if c.S3.Bucket == "" {
		return errors.New("s3 bucket is required")
	}

	// encryption password is required: without it the repositories are
	// unrecoverable, and auto-generation is a deliberate non-feature
	if c.Encryption.Password == "" {
		return errors.New("encryption password is required")
	}

	// temp path must exist (or be creatable) and be writable, since
	// restic restores larger than tmpfs are a known corruption source
	if c.TempPath == "" {
		return errors.New("temp path is required")
	}
	if err := ensureWritableDir(c.TempPath); err != nil {
		return fmt.Errorf("temp path %s: %w", c.TempPath, err)
	}

	// at least one backup entry, each fully specified, names unique
	if len(c.Backups) == 0 {
		return errors.New("at least one backup entry is required")
	}
	seen := make(map[string]bool, len(c.Backups))
	for i, b := range c.Backups {
		if b.Name == "" {
			return fmt.Errorf("backup %d: name is required", i+1)
		}
		if seen[b.Name] {
			return fmt.Errorf("backup name %q is used more than once", b.Name)
		}
		seen[b.Name] = true
		if b.StartPath == "" {
			return fmt.Errorf("backup %q: start path is required", b.Name)
		}
		if b.RetentionDays < 1 {
			return fmt.Errorf("backup %q: retention days must be at least 1", b.Name)
		}
	}

	return nil
}

// Save writes the configuration to path with restrictive permissions,
// creating the parent directory if needed.
func (c *Config) Save(path string) error {

	// serialize
	raw, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	// ensure the parent directory exists, owner-only
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// owner-only file perms: the config holds s3 credentials and the
	// repository encryption password
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}

// RepoURL builds the restic repository URL for the named backup,
// one repository per backup name under the configured bucket.
func (c *Config) RepoURL(name string) string {
	return fmt.Sprintf("s3:https://%s/%s/%s", c.S3.Endpoint, c.S3.Bucket, name)
}

// FindBackup returns the backup entry with the given name.
func (c *Config) FindBackup(name string) (Backup, bool) {
	for _, b := range c.Backups {
		if b.Name == name {
			return b, true
		}
	}
	return Backup{}, false
}

// Concurrency returns the number of backup entries to run in
// parallel: half the available CPUs, never less than one.
func Concurrency() int {
	n := runtime.NumCPU() / 2
	if n < 1 {
		n = 1
	}
	return n
}

// ensureWritableDir creates dir if missing and verifies it is
// writable by attempting to create and remove a probe file.
func ensureWritableDir(dir string) error {

	// create if needed
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// probe writability directly rather than trusting mode bits
	probe, err := os.CreateTemp(dir, ".kp-probe-*")
	if err != nil {
		return fmt.Errorf("not writable: %w", err)
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)

	return nil
}
