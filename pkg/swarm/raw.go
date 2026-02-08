package swarm

import (
	"encoding/binary"
	"errors"
	"time"
)

var (
	ErrMessageTruncated = errors.New("message truncated")
)

var networkByteOrder = binary.BigEndian

func copyBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

type (
	// Tuple is used in swarm protocol.
	// It is type alias which is asserted to [][]byte in "switch name.(type)" operator.
	Tuple = []Field
	// Field of the Tuple in swarm protocol.
	// It is type alias which is asserted to []byte in "switch name.(type)" operator.
	Field = []byte
)

const (
	TupleK1 = iota
	TupleK2
	TupleK3
	TupleV1
	TupleV2
	TupleV3
)

func UnpackTuplesList(body []byte) ([]Tuple, error) {
	bodySize := uint32(len(body))
	offset := uint32(0)
	if bodySize < offset+4 {
		return nil, ErrMessageTruncated
	}
	tupleCount := networkByteOrder.Uint32(body[offset:])
	offset += 4

	if tupleCount == 0 {
		return nil, nil
	}

	tupleList := make([]Tuple, tupleCount)

	for tupleIndex := uint32(0); tupleIndex < tupleCount; tupleIndex++ {
		if bodySize < offset+4 {
			return nil, ErrMessageTruncated
		}
		count := networkByteOrder.Uint32(body[offset:])
		offset += 4

		tupleList[tupleIndex] = make([]Field, count)
		var valueLen uint32

		for i := uint32(0); i < count; i++ {
			if bodySize < offset+4 {
				return nil, ErrMessageTruncated
			}

			valueLen = networkByteOrder.Uint32(body[offset:])
			offset += 4

			if bodySize < offset+valueLen {
				return nil, ErrMessageTruncated
			}
			tupleList[tupleIndex][i] = body[offset : offset+valueLen]
			offset += valueLen
		}
	}

	return tupleList, nil
}

func UnpackTuple(body []byte) ([]Field, error) {
	bodySize := uint32(len(body))
	offset := uint32(0)
	if bodySize < offset+4 {
		return nil, ErrMessageTruncated
	}
	count := networkByteOrder.Uint32(body[offset:])
	offset += 4

	tuple := make([]Field, count)
	var valueLen uint32

	for i := uint32(0); i < count; i++ {
		if bodySize < offset+4 {
			return nil, ErrMessageTruncated
		}

		valueLen = networkByteOrder.Uint32(body[offset:])
		offset += 4

		if bodySize < offset+valueLen {
			return nil, ErrMessageTruncated
		}
		tuple[i] = body[offset : offset+valueLen]
		offset += valueLen
	}

	return tuple, nil
}

func PackTuple(tuple []Field) []byte {
	bodySize := 4 + 4*len(tuple)
	count := len(tuple)
	for i := 0; i < count; i++ {
		bodySize += len(tuple[i])
	}

	body := make([]byte, bodySize)
	offset := 0
	networkByteOrder.PutUint32(body[offset:], uint32(count))
	offset += 4

	var valueLen int
	for i := 0; i < count; i++ {
		valueLen = len(tuple[i])
		networkByteOrder.PutUint32(body[offset:], uint32(valueLen))
		offset += 4
		copy(body[offset:], tuple[i])
		offset += valueLen
	}
	return body
}

func PackTuplesList(tuples []Tuple) []byte {
	// tuples count
	length := 4
	for _, tuple := range tuples {
		// tuple's fields count
		length += 4
		for _, field := range tuple {
			// field bytes count
			length += 4
			if len(field) > 0 {
				length += len(field)
			}
		}
	}
	body := make([]byte, length)
	offset := 0
	networkByteOrder.PutUint32(body[offset:], uint32(len(tuples)))
	offset += 4
	for _, tuple := range tuples {
		networkByteOrder.PutUint32(body[offset:], uint32(len(tuple)))
		offset += 4
		for _, field := range tuple {
			networkByteOrder.PutUint32(body[offset:], uint32(len(field)))
			offset += 4
			if len(field) > 0 {
				copy(body[offset:], field)
				offset += len(field)
			}
		}
	}
	return body
}

func EncodeUint32(value uint32) Field {
	body := make([]byte, 4)
	networkByteOrder.PutUint32(body, value)
	return body
}

func DecodeUint32(value []byte) uint32 {
	return networkByteOrder.Uint32(value)
}

var ZeroLength = EncodeLength(0)

func EncodeLength(length int) Field {
	return EncodeUint32(uint32(length))
}

func EncodeTime(ts time.Time) Field {
	return EncodeUint32(uint32(ts.Unix()))
}
