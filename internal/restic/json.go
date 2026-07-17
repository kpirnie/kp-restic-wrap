/**
 * KP - Restic Backup Wrapper
 *
 * Typed access to restic's JSON output.
 *
 * Copyright (c) 2026 Kevin Pirnie
 * Licensed under the MIT License. See LICENSE for details.
 */

package restic

import (
	"encoding/json"
	"fmt"
	"time"
)

// Snapshot is a single entry from `restic snapshots --json`.
type Snapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags"`
}

// Snapshots returns all snapshots in the repository, oldest first.
func (c *Client) Snapshots() ([]Snapshot, error) {

	out, err := c.Output("snapshots", "--json")
	if err != nil {
		return nil, err
	}

	var snaps []Snapshot
	if err := json.Unmarshal(out, &snaps); err != nil {
		return nil, fmt.Errorf("parsing snapshot list: %w", err)
	}

	return snaps, nil
}
