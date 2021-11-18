// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"fmt"
	"net/http"

	"github.com/ava-labs/avalanchego/utils/formatting"
)

// StaticService defines the base service for the timestamp vm
type StaticService struct{}

// CreateStaticService ...
func CreateStaticService() *StaticService {
	return &StaticService{}
}

// EncodeArgs are arguments for Encode
type EncodeArgs struct {
	Data     string              `json:"data"`
	Encoding formatting.Encoding `json:"encoding"`
	Length   int32               `json:"length"`
}

// EncodeReply is the reply from Encode
type EncodeReply struct {
	Bytes    string              `json:"bytes"`
	Encoding formatting.Encoding `json:"encoding"`
}

// Encode returns the encoded data
func (ss *StaticService) Encode(_ *http.Request, args *EncodeArgs, reply *EncodeReply) error {
	if len(args.Data) == 0 {
		return fmt.Errorf("argument Data cannot be empty")
	}
	var argBytes []byte
	if args.Length > 0 {
		argBytes = make([]byte, args.Length)
		copy(argBytes, args.Data)
	} else {
		argBytes = []byte(args.Data)
	}

	bytes, err := formatting.EncodeWithChecksum(args.Encoding, argBytes)
	if err != nil {
		return fmt.Errorf("couldn't encode data as string: %s", err)
	}
	reply.Bytes = bytes
	reply.Encoding = args.Encoding
	return nil
}

// DecodeArgs are arguments for Decode
type DecodeArgs struct {
	Bytes    string              `json:"bytes"`
	Encoding formatting.Encoding `json:"encoding"`
}

// DecodeReply is the reply from Decode
type DecodeReply struct {
	Data     string              `json:"data"`
	Encoding formatting.Encoding `json:"encoding"`
}

// Decode returns the Decoded data
func (ss *StaticService) Decode(_ *http.Request, args *DecodeArgs, reply *DecodeReply) error {
	bytes, err := formatting.Decode(args.Encoding, args.Bytes)
	if err != nil {
		return fmt.Errorf("couldn't Decode data as string: %s", err)
	}
	reply.Data = string(bytes)
	reply.Encoding = args.Encoding
	return nil
}
