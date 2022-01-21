// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

var (
	// Space Usage
	CountCreatedSpaces      = []byte("createdSpaces")
	CountExpiredSpaces      = []byte("expiredSpaces")
	CountLifelineUnitsSpent = []byte("lifelineUnitsSpent")

	// Key/Value Usage
	CountSetKeys      = []byte("setKeys")      // includes new sets + modifications
	CountSetValueSize = []byte("setValueSize") // includes set + modified values
	CountDeletedKeys  = []byte("deletedKeys")

	// Active State Usage
	CountActivePaths     = []byte("activePaths")
	CountActiveValueSize = []byte("activeValueSize")
	CountActiveUnits     = []byte("activeUnits")
)
