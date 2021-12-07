package bsoncv

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	Float64        = '\x01'
	String         = '\x02'
	Object         = '\x03'
	Array          = '\x04'
	ObjectId       = '\x07'
	Boolean        = '\x08'
	UnixTimeMillis = '\x09'
	Null           = '\x0A'
	Int32          = '\x10'
	Time           = '\x11'
	Int64          = '\x12'
	Dec128         = '\x13'
	Terminal       = '\x00'
	False          = '\x00'
	True           = '\x01'
)

func ToJson(bsonbytes []byte) []byte {
	if len(bsonbytes) == 0 {
		return bsonbytes
	}
	// from here it is assumed that the bson is valid
	initialCap := len(bsonbytes)
	if len(bsonbytes) > 1000000 {
		initialCap = 1000000
	}
	jsonbytes := make([]byte, 0, initialCap)
	idx := 4
	jsonbytes = append(jsonbytes, '{')

	// Max nesting depth is 64
	var stack [64]byte
	stackptr := 0
	stack[stackptr] = '}'

	for idx < len(bsonbytes) {

		switch bsonbytes[idx] {
		case Float64:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, '"', ':')
			}
			idx = end + 1
			jsonbytes = append(
				jsonbytes,
				[]byte(strconv.FormatFloat(
					math.Float64frombits(binary.LittleEndian.Uint64(bsonbytes[idx:idx+8])),
					'f', -1, 64),
				)...,
			)
			idx += 8
		case String:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			length := int(binary.LittleEndian.Uint32(bsonbytes[idx : idx+4]))
			idx += 4
			jsonbytes = append(jsonbytes, '"')
			for i := idx; i < idx+length-1; i++ {
				switch bsonbytes[i] {
				case '"':
					jsonbytes = append(jsonbytes, '\\', '"')
				case '\n':
					jsonbytes = append(jsonbytes, '\\', 'n')
				case '\t':
					jsonbytes = append(jsonbytes, '\\', 't')
				case '\\':
					jsonbytes = append(jsonbytes, '\\', '\\')
				case '\r':
					jsonbytes = append(jsonbytes, '\\', 'r')
				default:
					jsonbytes = append(jsonbytes, bsonbytes[i])
				}
			}
			jsonbytes = append(jsonbytes, '"')
			idx += length
		case Object:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			jsonbytes = append(jsonbytes, '{')
			stackptr++
			stack[stackptr] = '}'
			idx += 4 // this is an iterative solution so we can throw away the length
		case Array:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			jsonbytes = append(jsonbytes, '[')
			stackptr++
			stack[stackptr] = ']'

			idx += 4 // this is an iterative solution so we can throw away the length
		case ObjectId:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			id := hex.EncodeToString(bsonbytes[idx : idx+12])
			jsonbytes = append(jsonbytes, '"')
			jsonbytes = append(jsonbytes, id...)
			jsonbytes = append(jsonbytes, '"')
			idx += 12
		case Boolean:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			if bsonbytes[idx] == True {
				jsonbytes = append(jsonbytes, "true"...)
			} else {
				jsonbytes = append(jsonbytes, "false"...)
			}
			idx++
		case UnixTimeMillis:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element id information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			timestr := `"` + time.Unix(0, int64(binary.LittleEndian.Uint64(bsonbytes[idx:idx+8]))*1000000).Format(time.RFC3339Nano) + `"`
			jsonbytes = append(jsonbytes, timestr...)
			idx += 8
		case Null:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			jsonbytes = append(jsonbytes, "null"...)
		case Int32:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			jsonbytes = append(
				jsonbytes,
				[]byte(strconv.FormatUint(uint64(binary.LittleEndian.Uint32(bsonbytes[idx:idx+4])),
					10))...)
			idx += 4
		case Time:
			panic(jsonbytes)
		case Int64:
			idx++
			end := idx
			for bsonbytes[end] != Terminal {
				end++
			}
			if stack[stackptr] == '}' { // we skip the element mongo information in an array
				jsonbytes = append(jsonbytes, '"')
				jsonbytes = append(jsonbytes, bsonbytes[idx:end]...)
				jsonbytes = append(jsonbytes, "\":"...)
			}
			idx = end + 1
			jsonbytes = append(
				jsonbytes,
				[]byte(strconv.FormatUint(binary.LittleEndian.Uint64(bsonbytes[idx:idx+8]), 10))...)
			idx += 8
		case Dec128:
			panic(jsonbytes)
		case Terminal:
			idx++
			jsonbytes = append(jsonbytes, stack[stackptr])
			stack[stackptr] = Terminal
			stackptr--
		default:
			fmt.Println(bsonbytes[idx])
			return jsonbytes
		}
		// Add commas in the right spots
		if idx < len(bsonbytes) &&
			bsonbytes[idx] != Terminal &&
			jsonbytes[len(jsonbytes)-1] != '{' &&
			jsonbytes[len(jsonbytes)-1] != '[' {
			jsonbytes = append(jsonbytes, ',')
		}
	}
	return jsonbytes
}

