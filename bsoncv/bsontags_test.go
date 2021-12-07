package bsoncv_test

import (
	"encoding/json"
	"github.com/dustinevan/chron"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mongo/bsoncv"
	"reflect"
	"testing"
	"time"
)

type testCase struct {
	caseNum    int
	name       string
	testStruct interface{}
	expected   map[string]interface{}
}

var (
	objectId = primitive.ObjectID([12]byte{1, 35, 69, 103, 137, 171, 205, 239, 1, 35, 69, 103}) // lol 0123456789abcdef01234567
	cases    = [...]testCase{
		{
			caseNum:    1,
			name:       "It handles nil",
			testStruct: nil,
			expected:   nil,
		},
		{
			caseNum: 2,
			name:    "It handles zero values with no omitempty like this",
			testStruct: struct {
				ID   string
				Date time.Time
			}{},
			expected: map[string]interface{}{
				"ID":   "",
				"Date": time.Time{},
			},
		},
		{
			caseNum: 3,
			name:    "It works without bsoncv tags",
			testStruct: struct {
				ID   string
				Date time.Time
			}{
				ID:   "0123456789abcdef01234567",
				Date: chron.NewYear(2020).Time,
			},
			expected: map[string]interface{}{
				"ID":   "0123456789abcdef01234567",
				"Date": chron.NewYear(2020).Time,
			},
		},
		{
			caseNum: 4,
			name:    "It uses json tags for naming",
			testStruct: struct {
				ID   string    `json:"_id"`
				Date time.Time `json:"date"`
			}{
				ID:   "0123456789abcdef01234567",
				Date: chron.NewYear(2020).Time,
			},
			expected: map[string]interface{}{
				"_id":  "0123456789abcdef01234567",
				"date": chron.NewYear(2020).Time,
			},
		},
		{
			caseNum: 5,
			name:    "It prioritizes bson names over json names",
			testStruct: struct {
				ID   string    `json:"_id" bson:"_id1"`
				Date time.Time `json:"date"`
			}{
				ID:   "0123456789abcdef01234567",
				Date: chron.NewYear(2020).Time,
			},
			expected: map[string]interface{}{
				"_id1": "0123456789abcdef01234567",
				"date": chron.NewYear(2020).Time,
			},
		},
		{
			caseNum: 6,
			name:    "It prioritizes bsoncv names over bson names",
			testStruct: struct {
				ID   string    `json:"_id" bson:"_id1" bsoncv:"id3"`
				Date time.Time `json:"date"`
			}{
				ID:   "0123456789abcdef01234567",
				Date: chron.NewYear(2020).Time,
			},
			expected: map[string]interface{}{
				"id3":  "0123456789abcdef01234567",
				"date": chron.NewYear(2020).Time,
			},
		},
		{
			caseNum: 7,
			name:    "It converts strings to ObjectIDs",
			testStruct: struct {
				ID   string    `json:"_id" bson:"id3" bsoncv:",$oid"`
				Date time.Time `json:"date"`
			}{
				ID:   "0123456789abcdef01234567",
				Date: chron.NewYear(2020).Time,
			},
			expected: map[string]interface{}{
				"id3":  objectId,
				"date": chron.NewYear(2020).Time,
			},
		},
		{
			caseNum: 8,
			name:    "It converts ints to time.Time with millisecond precision",
			testStruct: struct {
				ID   string `json:"_id" bson:"id3" bsoncv:",$oid"`
				Date int    `bsoncv:"date,$date"`
			}{
				ID:   "0123456789abcdef01234567",
				Date: int(chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).UnixNano() / int64(time.Millisecond)),
			},
			expected: map[string]interface{}{
				"id3":  objectId,
				"date": chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).Time.Local(),
			},
		},
		{
			caseNum: 9,
			name:    "It converts strings to time.Time with default formats",
			testStruct: struct {
				ID    string `json:"_id" bson:"id3" bsoncv:",$oid"`
				Date1 string `bson:"date1" bsoncv:",$date,,"`
				Date2 string `bsoncv:"birthday,$date,,UnixDate"`
				Date3 string `bsoncv:"cardExp,$date,,01/06"`
			}{
				ID:    "0123456789abcdef01234567",
				Date1: chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).Time.Format(bsoncv.RFC3339Milli),
				Date2: chron.NewDay(2025, time.July, 14).Time.Format(time.UnixDate),
				Date3: chron.NewMonth(2022, time.March).Time.Format("01/06"),
			},

			expected: map[string]interface{}{
				"id3":      objectId,
				"date1":    chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).Time,
				"birthday": chron.NewDay(2025, time.July, 14).Time,
				"cardExp":  chron.NewMonth(2022, time.March).Time,
			},
		},
		{
			caseNum: 8,
			name:    "It inlines raw json",
			testStruct: struct {
				ID     string `json:"_id" bson:"id3" bsoncv:",$oid"`
				Date   int    `bsoncv:"date,$date"`
				Json   []byte `bsoncv:"msg,$json"`
				Number []byte `bsoncv:"num,$json"`
				String []byte `bsoncv:"str,$json"`
				Bool   []byte `bsoncv:"bool,$json"`
				Null   []byte `bsoncv:"nothing,$json"`
			}{
				ID:     "0123456789abcdef01234567",
				Date:   int(chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).UnixNano() / int64(time.Millisecond)),
				Json:   []byte(`{"text":"This is a message","meta":{"array":[0,1,"two",3.0,false,null],"data":{}}}`),
				Number: []byte(`1`),
				String: []byte(`"json"`),
				Bool:   []byte(`true`),
				Null:   []byte(`null`),
			},
			expected: map[string]interface{}{
				"id3":  objectId,
				"date": chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).Time.Local(),
				"msg": map[string]interface{}{
					"text": "This is a message",
					"meta": map[string]interface{}{
						"array": []interface{}{
							float64(0), float64(1), "two", 3.0, false, nil,
						},
						"data": map[string]interface{}{},
					},
				},
				"num":     float64(1),
				"str":     "json",
				"bool":    true,
				"nothing": nil,
			},
		},
		{
			caseNum: 9,
			name:    "It has cool raw json wrapper functionality",
			testStruct: struct {
				ID  string                   `bsoncv:"_id,$oid"`
				Msg CoolJSONWrapperShowOffer `bsoncv:"msg,$json"`
			}{
				ID: "0123456789abcdef01234567",
				Msg: CoolJSONWrapperShowOffer{
					Json: []byte(`{"text":"This is a message","meta":{"array":[0,1,"two",3.0,false,null],"data":{}}}`),
				},
			},
			expected: map[string]interface{}{
				"_id": objectId,
				"msg": map[string]interface{}{
					"text": "This is a message",
					"meta": map[string]interface{}{
						"array": []interface{}{
							float64(0), float64(1), "two", 3.0, false, nil,
						},
						"data": map[string]interface{}{},
					},
				},
			},
		},
		{
			caseNum: 10,
			name:    "It converts nested structs",
			testStruct: struct {
				ID     string `bsoncv:"_id,$oid"`
				Nested Nested `bsoncv:"nested"`
			}{
				ID:     "0123456789abcdef01234567",
				Nested: Nested{ID: "0123456789abcdef01234567"},
			},
			expected: map[string]interface{}{
				"_id": objectId,
				"nested": map[string]interface{}{
					"_id": objectId,
				},
			},
		},
		{
			caseNum: 11,
			name:    "It omits empty",
			testStruct: struct {
				ID         string `bsoncv:"_id,$oid,omitempty"`
				IntDate1   int    `bsoncv:"intDate1,$date"`
				IntDate2   int    `bsoncv:"intDate2,$date,omitempty"`
				StringDate string `bsoncv:"strDate,$date,omitempty"`
				Msg1       []byte `bsoncv:"msg1,$json"`
				Msg2       []byte `bsoncv:"msg2,$json,omitempty"`
				Msg3       []byte `bsoncv:"msg3,$json,omitempty"`
			}{
				ID:         "",
				IntDate1:   0,
				IntDate2:   0,
				StringDate: "",
				Msg1:       []byte{},
				Msg2:       []byte{},
				Msg3:       nil,
			},
			expected: map[string]interface{}{
				"intDate1": time.Time{},
				"msg1":     nil,
			},
		},
		{
			caseNum: 12,
			name:    "It converts underlying pointer values and omits empty",
			testStruct: struct {
				ID1     *string                   `bsoncv:"id1,$oid,omitempty"`
				ID2     *string                   `bsoncv:"id2,$oid,omitempty"`
				Date1   *int                      `bsoncv:"date1,$date,omitempty"`
				Date2   *int                      `bsoncv:"date2,$date,omitempty"`
				Date3   *string                   `bsoncv:"date3,$date,omitempty,UnixDate"`
				Date4   *string                   `bsoncv:"date4,$date,omitempty,UnixDate"`
				Nested1 *Nested                   `bsoncv:"nested1,,omitempty"`
				Nested2 *Nested                   `bsoncv:"nested2,,omitempty"`
				Cool1   *CoolJSONWrapperShowOffer `bsoncv:"cool1,$json,omitempty"`
				Cool2   *CoolJSONWrapperShowOffer `bsoncv:"cool2,$json,omitempty"`
			}{
				ID1:     stringPtr("0123456789abcdef01234567"),
				ID2:     nil,
				Date1:   intPtr(int(chron.NewYear(2020).UnixNano() / int64(time.Millisecond))),
				Date2:   nil,
				Date3:   stringPtr(chron.NewYear(2021).Time.Format(time.UnixDate)),
				Date4:   nil,
				Nested1: &Nested{ID: "0123456789abcdef01234567"},
				Nested2: nil,
				Cool1: &CoolJSONWrapperShowOffer{
					Json: []byte(`{"text":"This is a message","meta":{"array":[0,1,"two",3.0,false,null],"data":{}}}`),
				},
				Cool2: nil,
			},
			expected: map[string]interface{}{
				"id1":   objectId,
				"date1": chron.NewYear(2020).Time.Local(),
				"date3": chron.NewYear(2021).Time,
				"nested1": map[string]interface{}{
					"_id": objectId,
				},
				"cool1": map[string]interface{}{
					"text": "This is a message",
					"meta": map[string]interface{}{
						"array": []interface{}{
							float64(0), float64(1), "two", 3.0, false, nil,
						},
						"data": map[string]interface{}{},
					},
				},
			},
		},
		{
			caseNum: 13,
			name:    "it does no conversion when none is needed",
			testStruct: struct {
				AString string `bsoncv:"aString,,omitempty"`
				ADate   string `bsoncv:"aDate,,omitempty"`
			}{
				AString: "0123456789abcdef01234567",
				ADate: chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).
					Time.Local().Format(bsoncv.RFC3339Milli),
			},
			expected: map[string]interface{}{
				"aString": "0123456789abcdef01234567",
				"aDate": chron.NewMilli(2020, time.January, 13, 11, 32, 13, 222).
					Time.Local().Format(bsoncv.RFC3339Milli),
			},
		},
	}
)

func TestStructToMap(t *testing.T) {
	for _, c := range cases {
		actual, err := bsoncv.StructToMap(c.testStruct)
		if err != nil {
			t.Errorf("%+v", err)
		}
		if !reflect.DeepEqual(c.expected, actual) {
			t.Errorf("FAILED: caseNum:%v - %s\nexpected: %v\nactual:   %v\n", c.caseNum, c.name, c.expected, actual)
		}
	}
}

func TestToBson(t *testing.T) {
	for _, c := range cases {
		bsn, err := bsoncv.ToBson(c.testStruct)
		if err != nil {
			t.Error(err)
		}
		t.Logf(string(bsoncv.ToJson(bsn)))
		jsn, err := json.Marshal(c.expected)
		if err != nil {
			t.Error(err)
		}
		t.Logf(string(jsn))
	}
}

type CoolJSONWrapperShowOffer struct {
	Json []byte
}

func (c CoolJSONWrapperShowOffer) JsonBytes() []byte {
	return c.Json
}

type Nested struct {
	ID string `bsoncv:"_id,$oid"`
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
