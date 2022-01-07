package chain

// Units is the "cost" of a value
func Units(b []byte) int64 {
	return int64(len(b)/ValueUnitLength + 1)
}
