package tdata

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// Type is the inner type of an EIP-712 message
type Type struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (t *Type) isArray() bool {
	return strings.HasSuffix(t.Type, "[]")
}

// typeName returns the canonical name of the type. If the type is 'Person[]', then
// this method returns 'Person'
func (t *Type) typeName() string {
	if strings.HasSuffix(t.Type, "[]") {
		return strings.TrimSuffix(t.Type, "[]")
	}
	return t.Type
}

func (t *Type) isReferenceType() bool {
	if len(t.Type) == 0 {
		return false
	}
	// Reference types must have a leading uppercase character
	r, _ := utf8.DecodeRuneInString(t.Type)
	return unicode.IsUpper(r)
}

type Types map[string][]Type

type TypedDataMessage = map[string]interface{}

// TypedDataDomain represents the domain part of an EIP-712 message.
// https://github.com/ethereum/go-ethereum/blob/619a3e70858e60240ce1df75bdf65ba748387e57/signer/core/apitypes/types.go#L246
type TypedDataDomain struct {
	Name  string `json:"name"`
	Magic uint64 `json:"magic"`
}

type TypedData struct {
	Types       Types            `json:"types"`
	PrimaryType string           `json:"primaryType"`
	Domain      TypedDataDomain  `json:"domain"`
	Message     TypedDataMessage `json:"message"`
}

var (
	EIP712Domain = []Type{
		{Name: "name", Type: "string"},
		{Name: "magic", Type: "uint64"},
	}
)

func spacesDomain(m uint64) TypedDataDomain {
	return TypedDataDomain{
		Name:  "Spaces",
		Magic: m,
	}
}

func CreateTypedData(magic uint64, txType string, txFields []Type, msg TypedDataMessage) *TypedData {
	return &TypedData{
		Types: Types{
			txType:         txFields,
			"EIP712Domain": EIP712Domain,
		},
		PrimaryType: txType,
		Domain:      spacesDomain(magic),
		Message:     msg,
	}
}

func DigestHash(td *TypedData) ([]byte, error) {
	typedDataHash, err := td.HashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return nil, err
	}
	domainSeparator, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return nil, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256Hash(rawData).Bytes(), nil
}

// HashStruct generates a keccak256 hash of the encoding of the provided data
func (typedData *TypedData) HashStruct(primaryType string, data TypedDataMessage) (hexutil.Bytes, error) {
	encodedData, err := typedData.EncodeData(primaryType, data, 1)
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(encodedData), nil
}

// Dependencies returns an array of custom types ordered by their hierarchical reference tree
func (typedData *TypedData) Dependencies(primaryType string, found []string) []string {
	includes := func(arr []string, str string) bool {
		for _, obj := range arr {
			if obj == str {
				return true
			}
		}
		return false
	}

	if includes(found, primaryType) {
		return found
	}
	if typedData.Types[primaryType] == nil {
		return found
	}
	found = append(found, primaryType)
	for _, field := range typedData.Types[primaryType] {
		for _, dep := range typedData.Dependencies(field.Type, found) {
			if !includes(found, dep) {
				found = append(found, dep)
			}
		}
	}
	return found
}

// EncodeType generates the following encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func (typedData *TypedData) EncodeType(primaryType string) hexutil.Bytes {
	// Get dependencies primary first, then alphabetical
	deps := typedData.Dependencies(primaryType, []string{})
	if len(deps) > 0 {
		slicedDeps := deps[1:]
		sort.Strings(slicedDeps)
		deps = append([]string{primaryType}, slicedDeps...)
	}

	// Format as a string with fields
	var buffer bytes.Buffer
	for _, dep := range deps {
		buffer.WriteString(dep)
		buffer.WriteString("(")
		for _, obj := range typedData.Types[dep] {
			buffer.WriteString(obj.Type)
			buffer.WriteString(" ")
			buffer.WriteString(obj.Name)
			buffer.WriteString(",")
		}
		buffer.Truncate(buffer.Len() - 1)
		buffer.WriteString(")")
	}
	return buffer.Bytes()
}

// TypeHash creates the keccak256 hash  of the data
func (typedData *TypedData) TypeHash(primaryType string) hexutil.Bytes {
	return crypto.Keccak256(typedData.EncodeType(primaryType))
}

// EncodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func (typedData *TypedData) EncodeData(primaryType string, data map[string]interface{}, depth int) (hexutil.Bytes, error) {
	// Assume provided data is valid
	buffer := bytes.Buffer{}

	// Verify extra data
	if exp, got := len(typedData.Types[primaryType]), len(data); exp < got {
		return nil, fmt.Errorf("there is extra data provided in the message (%d < %d)", exp, got)
	}

	// Add typehash
	buffer.Write(typedData.TypeHash(primaryType))

	// Add field contents. Structs and arrays have special handlers.
	for _, field := range typedData.Types[primaryType] {
		encType := field.Type
		encValue := data[field.Name]
		if encType[len(encType)-1:] == "]" {
			arrayValue, ok := encValue.([]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}

			arrayBuffer := bytes.Buffer{}
			parsedType := strings.Split(encType, "[")[0]
			for _, item := range arrayValue {
				if typedData.Types[parsedType] != nil {
					mapValue, ok := item.(map[string]interface{})
					if !ok {
						return nil, dataMismatchError(parsedType, item)
					}
					encodedData, err := typedData.EncodeData(parsedType, mapValue, depth+1)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(encodedData)
				} else {
					bytesValue, err := typedData.EncodePrimitiveValue(parsedType, item, depth)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(bytesValue)
				}
			}

			buffer.Write(crypto.Keccak256(arrayBuffer.Bytes()))
		} else if typedData.Types[field.Type] != nil {
			mapValue, ok := encValue.(map[string]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}
			encodedData, err := typedData.EncodeData(field.Type, mapValue, depth+1)
			if err != nil {
				return nil, err
			}
			buffer.Write(crypto.Keccak256(encodedData))
		} else {
			byteValue, err := typedData.EncodePrimitiveValue(encType, encValue, depth)
			if err != nil {
				return nil, err
			}
			buffer.Write(byteValue)
		}
	}
	return buffer.Bytes(), nil
}

// Attempt to parse bytes in different formats: byte array, hex string, hexutil.Bytes.
func parseBytes(encType interface{}) ([]byte, bool) {
	switch v := encType.(type) {
	case []byte:
		return v, true
	case hexutil.Bytes:
		return v, true
	case string:
		bytes, err := hexutil.Decode(v)
		if err != nil {
			return nil, false
		}
		return bytes, true
	default:
		return nil, false
	}
}

func parseInteger(encType string, encValue interface{}) (*big.Int, error) {
	var (
		length int
		signed = strings.HasPrefix(encType, "int")
		b      *big.Int
	)
	if encType == "int" || encType == "uint" {
		length = 256
	} else {
		lengthStr := ""
		if strings.HasPrefix(encType, "uint") {
			lengthStr = strings.TrimPrefix(encType, "uint")
		} else {
			lengthStr = strings.TrimPrefix(encType, "int")
		}
		atoiSize, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid size on integer: %v", lengthStr)
		}
		length = atoiSize
	}
	switch v := encValue.(type) {
	case *math.HexOrDecimal256:
		b = (*big.Int)(v)
	case string:
		var hexIntValue math.HexOrDecimal256
		if err := hexIntValue.UnmarshalText([]byte(v)); err != nil {
			return nil, err
		}
		b = (*big.Int)(&hexIntValue)
	case float64:
		// JSON parses non-strings as float64. Fail if we cannot
		// convert it losslessly
		if float64(int64(v)) == v {
			b = big.NewInt(int64(v))
		} else {
			return nil, fmt.Errorf("invalid float value %v for type %v", v, encType)
		}
	}
	if b == nil {
		return nil, fmt.Errorf("invalid integer value %v/%v for type %v", encValue, reflect.TypeOf(encValue), encType)
	}
	if b.BitLen() > length {
		return nil, fmt.Errorf("integer larger than '%v'", encType)
	}
	if !signed && b.Sign() == -1 {
		return nil, fmt.Errorf("invalid negative value for unsigned type %v", encType)
	}
	return b, nil
}

// EncodePrimitiveValue deals with the primitive values found
// while searching through the typed data
func (typedData *TypedData) EncodePrimitiveValue(encType string, encValue interface{}, depth int) ([]byte, error) {
	switch encType {
	case "address":
		stringValue, ok := encValue.(string)
		if !ok || !common.IsHexAddress(stringValue) {
			return nil, dataMismatchError(encType, encValue)
		}
		retval := make([]byte, 32)
		copy(retval[12:], common.HexToAddress(stringValue).Bytes())
		return retval, nil
	case "bool":
		boolValue, ok := encValue.(bool)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		if boolValue {
			return math.PaddedBigBytes(common.Big1, 32), nil
		}
		return math.PaddedBigBytes(common.Big0, 32), nil
	case "string":
		strVal, ok := encValue.(string)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.Keccak256([]byte(strVal)), nil
	case "bytes":
		bytesValue, ok := parseBytes(encValue)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.Keccak256(bytesValue), nil
	}
	if strings.HasPrefix(encType, "bytes") {
		lengthStr := strings.TrimPrefix(encType, "bytes")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid size on bytes: %v", lengthStr)
		}
		if length < 0 || length > 32 {
			return nil, fmt.Errorf("invalid size on bytes: %d", length)
		}
		if byteValue, ok := parseBytes(encValue); !ok || len(byteValue) != length {
			return nil, dataMismatchError(encType, encValue)
		} else {
			// Right-pad the bits
			dst := make([]byte, 32)
			copy(dst, byteValue)
			return dst, nil
		}
	}
	if strings.HasPrefix(encType, "int") || strings.HasPrefix(encType, "uint") {
		b, err := parseInteger(encType, encValue)
		if err != nil {
			return nil, err
		}
		return math.U256Bytes(b), nil
	}
	return nil, fmt.Errorf("unrecognized type '%s'", encType)

}

// dataMismatchError generates an error for a mismatch between
// the provided type and data
func dataMismatchError(encType string, encValue interface{}) error {
	return fmt.Errorf("provided data '%v' doesn't match type '%s'", encValue, encType)
}

// Map generates a map version of the typed data
func (typedData *TypedData) Map() map[string]interface{} {
	dataMap := map[string]interface{}{
		"types":       typedData.Types,
		"domain":      typedData.Domain.Map(),
		"primaryType": typedData.PrimaryType,
		"message":     typedData.Message,
	}
	return dataMap
}

// Checks if the primitive value is valid
func isPrimitiveTypeValid(primitiveType string) bool {
	if primitiveType == "address" ||
		primitiveType == "address[]" ||
		primitiveType == "bool" ||
		primitiveType == "bool[]" ||
		primitiveType == "string" ||
		primitiveType == "string[]" {
		return true
	}
	if primitiveType == "bytes" ||
		primitiveType == "bytes[]" ||
		primitiveType == "bytes1" ||
		primitiveType == "bytes1[]" ||
		primitiveType == "bytes2" ||
		primitiveType == "bytes2[]" ||
		primitiveType == "bytes3" ||
		primitiveType == "bytes3[]" ||
		primitiveType == "bytes4" ||
		primitiveType == "bytes4[]" ||
		primitiveType == "bytes5" ||
		primitiveType == "bytes5[]" ||
		primitiveType == "bytes6" ||
		primitiveType == "bytes6[]" ||
		primitiveType == "bytes7" ||
		primitiveType == "bytes7[]" ||
		primitiveType == "bytes8" ||
		primitiveType == "bytes8[]" ||
		primitiveType == "bytes9" ||
		primitiveType == "bytes9[]" ||
		primitiveType == "bytes10" ||
		primitiveType == "bytes10[]" ||
		primitiveType == "bytes11" ||
		primitiveType == "bytes11[]" ||
		primitiveType == "bytes12" ||
		primitiveType == "bytes12[]" ||
		primitiveType == "bytes13" ||
		primitiveType == "bytes13[]" ||
		primitiveType == "bytes14" ||
		primitiveType == "bytes14[]" ||
		primitiveType == "bytes15" ||
		primitiveType == "bytes15[]" ||
		primitiveType == "bytes16" ||
		primitiveType == "bytes16[]" ||
		primitiveType == "bytes17" ||
		primitiveType == "bytes17[]" ||
		primitiveType == "bytes18" ||
		primitiveType == "bytes18[]" ||
		primitiveType == "bytes19" ||
		primitiveType == "bytes19[]" ||
		primitiveType == "bytes20" ||
		primitiveType == "bytes20[]" ||
		primitiveType == "bytes21" ||
		primitiveType == "bytes21[]" ||
		primitiveType == "bytes22" ||
		primitiveType == "bytes22[]" ||
		primitiveType == "bytes23" ||
		primitiveType == "bytes23[]" ||
		primitiveType == "bytes24" ||
		primitiveType == "bytes24[]" ||
		primitiveType == "bytes25" ||
		primitiveType == "bytes25[]" ||
		primitiveType == "bytes26" ||
		primitiveType == "bytes26[]" ||
		primitiveType == "bytes27" ||
		primitiveType == "bytes27[]" ||
		primitiveType == "bytes28" ||
		primitiveType == "bytes28[]" ||
		primitiveType == "bytes29" ||
		primitiveType == "bytes29[]" ||
		primitiveType == "bytes30" ||
		primitiveType == "bytes30[]" ||
		primitiveType == "bytes31" ||
		primitiveType == "bytes31[]" ||
		primitiveType == "bytes32" ||
		primitiveType == "bytes32[]" {
		return true
	}
	if primitiveType == "int" ||
		primitiveType == "int[]" ||
		primitiveType == "int8" ||
		primitiveType == "int8[]" ||
		primitiveType == "int16" ||
		primitiveType == "int16[]" ||
		primitiveType == "int32" ||
		primitiveType == "int32[]" ||
		primitiveType == "int64" ||
		primitiveType == "int64[]" ||
		primitiveType == "int128" ||
		primitiveType == "int128[]" ||
		primitiveType == "int256" ||
		primitiveType == "int256[]" {
		return true
	}
	if primitiveType == "uint" ||
		primitiveType == "uint[]" ||
		primitiveType == "uint8" ||
		primitiveType == "uint8[]" ||
		primitiveType == "uint16" ||
		primitiveType == "uint16[]" ||
		primitiveType == "uint32" ||
		primitiveType == "uint32[]" ||
		primitiveType == "uint64" ||
		primitiveType == "uint64[]" ||
		primitiveType == "uint128" ||
		primitiveType == "uint128[]" ||
		primitiveType == "uint256" ||
		primitiveType == "uint256[]" {
		return true
	}
	return false
}

// Map is a helper function to generate a map version of the domain
func (domain *TypedDataDomain) Map() map[string]interface{} {
	return map[string]interface{}{
		"name":  domain.Name,
		"magic": domain.Magic,
	}
}
