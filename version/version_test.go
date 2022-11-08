package version

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type rpcChainCompatibility struct {
	RPCChainVMProtocolVersion map[string]uint `json:"rpcChainVMProtocolVersion"`
}

const compatibilityFile = "../compatibility.json"

func TestCompatibility(t *testing.T) {
	compat, err := os.ReadFile(compatibilityFile)
	assert.NoError(t, err)

	var parsedCompat rpcChainCompatibility
	err = json.Unmarshal(compat, &parsedCompat)
	assert.NoError(t, err)

	_, valueInJSON := parsedCompat.RPCChainVMProtocolVersion[Version.String()]
	assert.True(t, valueInJSON)
}
