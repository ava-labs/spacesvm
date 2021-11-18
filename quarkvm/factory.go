// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/vms"
)

var _ vms.Factory = &Factory{}

// Factory ...
type Factory struct{}

// New ...
func (f *Factory) New(*snow.Context) (interface{}, error) { return &VM{}, nil }
