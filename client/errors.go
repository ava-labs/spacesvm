// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import "errors"

var ErrIntegrityFailure = errors.New("received file that does not match hash")
