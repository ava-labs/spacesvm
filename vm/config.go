// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"fmt"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/ids"
)

type Config struct {
	BuildInterval    time.Duration `serialize:"true" json:"buildInterval"`
	GossipInterval   time.Duration `serialize:"true" json:"gossipInterval"`
	RegossipInterval time.Duration `serialize:"true" json:"regossipInterval"`

	PruneLimit        int           `serialize:"true" json:"pruneLimit"`
	PruneInterval     time.Duration `serialize:"true" json:"pruneInterval"`
	FullPruneInterval time.Duration `serialize:"true" json:"fullPruneInterval"`

	CompactInterval time.Duration `serialize:"true" json:"compactInterval"`

	MempoolSize       int `serialize:"true" json:"mempoolSize"`
	ActivityCacheSize int `serialize:"true" json:"activityCacheSize"`

	StateSyncEnabled bool   `serialize:"true" json:"stateSyncEnabled"`
	StateSyncIDs     string `serialize:"true" json:"stateSyncIDs"`
}

func (c *Config) SetDefaults() {
	c.BuildInterval = 500 * time.Millisecond
	c.GossipInterval = 1 * time.Second
	c.RegossipInterval = 30 * time.Second

	c.PruneLimit = 128
	c.PruneInterval = time.Minute
	c.FullPruneInterval = time.Second

	c.CompactInterval = 1 * time.Minute

	c.MempoolSize = 1024
	c.ActivityCacheSize = 128
}

func (c *Config) StateSyncNodeIDs() ([]ids.NodeID, error) {
	// parse nodeIDs from state sync IDs in vm config
	var stateSyncIDs []ids.NodeID
	if c.StateSyncEnabled && len(c.StateSyncIDs) > 0 {
		nodeIDs := strings.Split(c.StateSyncIDs, ",")
		stateSyncIDs = make([]ids.NodeID, len(nodeIDs))
		for i, nodeIDString := range nodeIDs {
			nodeID, err := ids.NodeIDFromString(nodeIDString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s as NodeID: %w", nodeIDString, err)
			}
			stateSyncIDs[i] = nodeID
		}
	}
	return stateSyncIDs, nil
}
