package proto

import "encoding/binary"

// EncodeVarint encodes a uint64 into protobuf varint format.
func EncodeVarint(val uint64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, val)
	return buf[:n]
}

// EncodeTag encodes a field number and wire type.
func EncodeTag(fieldNum int, wireType int) []byte {
	return EncodeVarint(uint64((fieldNum << 3) | (wireType & 0x7)))
}

// EncodeStringField encodes a string field (wire type 2).
func EncodeStringField(fieldNum int, str string) []byte {
	if str == "" {
		return nil
	}
	tag := EncodeTag(fieldNum, 2)
	strBytes := []byte(str)
	length := EncodeVarint(uint64(len(strBytes)))
	res := append(tag, length...)
	return append(res, strBytes...)
}

// EncodeInt64Field encodes an int64 field (wire type 0).
func EncodeInt64Field(fieldNum int, val int64) []byte {
	if val == 0 {
		return nil
	}
	tag := EncodeTag(fieldNum, 0)
	return append(tag, EncodeVarint(uint64(val))...)
}

// EncodeBoolField encodes a boolean field (wire type 0).
func EncodeBoolField(fieldNum int, val bool) []byte {
	if !val {
		return nil
	}
	tag := EncodeTag(fieldNum, 0)
	return append(tag, 1)
}
