// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

var (
	// Space Usage
	CountCreatedSpaces      = []byte("createdSpaces")
	CountExpiredSpaces      = []byte("expiredSpaces")
	CountLifelineUnitsSpent = []byte("lifelineUnitsSpent")

	// Key/Value Usage
	CountPathCreated         = []byte("pathCreated")
	CountPathModified        = []byte("pathModified")
	CountPathUploadValueSize = []byte("pathUploadValueSize") // includes set + modified values

	// Active State Usage
	CountActivePaths     = []byte("activePaths")
	CountActiveValueSize = []byte("activeValueSize")
)
