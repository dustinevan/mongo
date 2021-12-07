package bsoncv

import (
	jsondec "encoding/json"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

// bsoncv Struct Tags are formatted like this:
// bsoncv:"fieldname,conversionType,omitempty,dateformat"
// if an element isn't specified the commas must be present.
// Example:
// type User struct {
// 	// *** Element Names ***
// 	// e_name: id1, valueType: bsontype.String
// 	ID1 string
// 	// e_name: _id, valueType: bsontype.String
// 	ID2 string  `json:"_id"`
// 	// e_name: _id2, valueType: bsontype.String
// 	ID3 string  `json:"_id1" bson:"_id2"`
// 	// e_name: _id4, valueType: bsontype.String
// 	ID4 string  `json:"_id2" bson:"_id3" bsoncv:"_id4"`
//
// 	// *** ObjectIDs ***
// 	// e_name: _id5, valueType: bsontype.ObjectID
// 	ID5 string  `bson:"_id5 bsoncv:,$oid"`
// 	// e_name: linkedId, valueType: bsontype.ObjectID, omitted if LinkedID == ""
// 	LinkedId string `bsoncv:"linkedId,$oid,omitemtpy"`
// 	// e_name: bsonOmitEmpty, valueType: bsontype.ObjectID, omitted if BsonOmitEmpty == ""
// 	BsonOmitEmpty string `bson:",omitempty" bsoncv:",$oid"`
// 	// e_name: msg, valueType: string, but omitted if UseCommas == ""
// 	UseCommas string `bson:"msg" bsoncv:",,omitempty"`
//
// 	// *** Dates ***
// 	// e_name: date1, valueType: bsontype.DateTime
// 	Date1 time.Time
// 	// e_name: birthday, valueType: bsontype.DateTime
// 	// conversion uses time.Unix(Date2/1000, Date2%1000) to match MongoDB's millisecond time denomination.
// 	Date2 int `json:"birthday" bsoncv:",$date"`
// 	// e_name: date3, valueType: bsontype.DateTime
// 	// conversion uses time.RFC1123Z. All const formats specified in time/format.go are supported
//  // including the custom format RFC3339Milli = "2006-01-02T15:04:05.000Z07:00" which is
//  // used as the default format
// 	// Note that commas must be used to skip the omitempty location
// 	Date3 string `bsoncv:"date3,$date,,RFC1123Z"`
// 	// e_name: ccExpDate, valueType: bsontype.DateTime
// 	// conversion uses the format specified in the tag
// 	// NOTE: No commas can be used in this specified format
// 	CustomDate string `bsoncv:"ccExpDate,$date,omitempty,01/02"`
// 	// e_name: ptr, valueType: bsontype.ObjectID
// 	// omitempty if it's nil
//
// 	// *** Pointers ***
// 	Pointer *string `bsoncv:"ptr,$oid,omitempty"`
// 	// e_name: ptr2, valueType: bsontype.ObjectID || bsontype.Null if Pointer2 == nil
// 	Pointer2 *string `bsoncv:"ptr2,$oid"`
//
// 	// *** Unstructured JSON ***
// 	// e_name: raw, the data is unmarshalled to an interface{} and the bson marshaller
// 	// works normally.
// 	RawJson []byte `bsoncv:"raw,$jsonbytes"`
// }

type convType int

const (
	invalid convType = iota
	oid
	date
	json
)

var convTypeNames = [...]string{
	"",
	"$oid",
	"$date",
	"$json",
}

func parseConvType(t string) convType {
	for i, name := range convTypeNames {
		if name == t {
			return convType(i)
		}
	}
	return invalid
}

type bsonConvTag struct {
	conv      convType
	omitempty bool
	datefmt   string
}

func parseBsonConvTag(tag string) bsonConvTag {
	parts := strings.Split(tag, ",")
	var t bsonConvTag
	if len(parts) > 1 {
		t.conv = parseConvType(parts[1])
	}
	if len(parts) > 2 {
		if parts[2] != "" {
			t.omitempty = true
		}
	}
	if len(parts) > 3 {
		if t.conv == date {
			if f, ok := timeFormats[parts[3]]; ok {
				t.datefmt = f
			} else {
				t.datefmt = parts[3]
			}
		}
	}
	return t
}

func (b bsonConvTag) convertString(v string) (interface{}, error) {
	if b.conv == oid {
		return primitive.ObjectIDFromHex(v)
	}
	if b.conv == date {
		fmt := RFC3339Milli
		if b.datefmt != "" {
			if tfmt, ok := timeFormats[b.datefmt]; ok {
				fmt = tfmt
			} else {
				fmt = b.datefmt
			}
		}
		return time.Parse(fmt, v)
	}
	return v, nil
}

func (b bsonConvTag) convertToTime(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return time.Unix(v/1000, v%1000*int64(time.Millisecond))
}

func (b bsonConvTag) convertJSONBytes(v []byte) (interface{}, error) {
	var i interface{}
	if len(v) == 0 {
		return i, nil
	}
	err := jsondec.Unmarshal(v, &i)
	return i, err
}

func StructToMap(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, nil
	}
	data := make(map[string]interface{})

	typ := reflect.TypeOf(v)
	value := reflect.ValueOf(v)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		name := fieldName(field)
		// omit this field
		if name == "-" {
			continue
		}
		tag := parseBsonConvTag(field.Tag.Get("bsoncv"))
		fieldValue := value.Field(i)
		if fieldValue.Kind() == reflect.Ptr {
			fieldValue = fieldValue.Elem()
		}

		switch fieldValue.Kind() {
		case reflect.String:
			if tag.conv != invalid {
				fv := fieldValue.String()
				if fv != "" || !tag.omitempty {
					value, err := tag.convertString(fv)
					if err != nil {
						return data, errors.Wrapf(err,
							"bsoncv failed to convert string |%s| to %s for field %s",
							fv, convTypeNames[tag.conv], name)
					}
					data[name] = value
				}
			} else {
				data[name] = fieldValue.Interface()
			}
		case reflect.Int, reflect.Int64:
			if tag.conv == date {
				fv := fieldValue.Int()
				if fv != 0 || !tag.omitempty {
					data[name] = tag.convertToTime(fv)
				}
			} else {
				data[name] = fieldValue.Interface()
			}
		case reflect.Slice:
			if tag.conv == json {
				fv := fieldValue.Interface()
				if fv == nil || !tag.omitempty {
					if bytes, ok := fv.([]byte); ok {
						if len(bytes) > 0 || !tag.omitempty {
							jsonGoInterfaces, err := tag.convertJSONBytes(bytes)
							if err != nil {
								return data, errors.Wrapf(err,
									"bsoncv failed to convert jsonbytes %s for field %s",
									string(bytes), name)
							}
							data[name] = jsonGoInterfaces
						}
					}
				}
			}
		case reflect.Struct:
			if tag.conv == json {
				if wrapper, ok := fieldValue.Interface().(jsonWrapper); ok {
					jsonGoInterfaces, err := tag.convertJSONBytes(wrapper.JsonBytes())
					if err != nil {
						return data, errors.Wrapf(err,
							"bsoncv failed to convert jsonbytes %s for field %s",
							string(wrapper.JsonBytes()), name)
					}
					data[name] = jsonGoInterfaces
				}
			} else if _, ok := fieldValue.Interface().(time.Time); ok {
				data[name] = fieldValue.Interface()
			} else {
				str, err := StructToMap(fieldValue.Interface())
				if err != nil {
					return data, err
				}
				data[name] = str
			}
		case reflect.Invalid:
			if !tag.omitempty {
				data[name] = nil
			}
		default:
			data[name] = fieldValue.Interface()
		}
	}
	return data, nil
}

func ToBson(v interface{}) ([]byte, error) {
	data, err := StructToMap(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert struct to map")
	}
	return bson.Marshal(data)
}

// Returns the field name to be used as the e_name in the bson spec.
// This order of priority is used:
// 1. alias name in the bsoncv tag
// 2. alias name in the bson tag
// 3. alias name in the json tag
// 4. the field name
// if bsoncv, or bson tags are "-" "" is returned
func fieldName(f reflect.StructField) string {
	// note that this is in priority order, the later tags override the earlier ones
	tagsToCheck := []string{"json", "bson", "bsoncv"}

	name := f.Name
	for _, key := range tagsToCheck {
		if b := f.Tag.Get(key); b != "" {
			if components := strings.Split(b, ","); len(components) > 0 {
				if n := strings.TrimSpace(components[0]); n != "" {
					// don't omit from bsoncv if json is '-'
					if !(key == "json" && n == "-") {
						name = n
					}
				}
			}
		}
	}
	return name
}

var timeFormats = map[string]string{
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
}

const RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

type jsonWrapper interface {
	// There is no error here because implementers should actually contain raw json
	// rather than simply marshalling to json. Valid json is expected.
	JsonBytes() []byte
}
