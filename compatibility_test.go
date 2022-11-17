package vm

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/ava-labs/spacesvm/version"
	"github.com/stretchr/testify/assert"
)

type rpcChainCompatibility struct {
	RPCChainVMProtocolVersion map[string]uint `json:"rpcChainVMProtocolVersion"`
}

//go:embed compatibility.json
var rpcChainVMProtocolCompatibilityBytes []byte

func TestCompatibility(t *testing.T) {
	var compat rpcChainCompatibility
	err := json.Unmarshal(rpcChainVMProtocolCompatibilityBytes, &compat)
	assert.NoError(t, err)

	_, valueInJSON := compat.RPCChainVMProtocolVersion[version.Version.String()]
	assert.True(t, valueInJSON)
}
