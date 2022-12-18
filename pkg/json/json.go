package json

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
)

type JsonObject map[string]interface{}

type JsonArray struct {
	array []interface{}
}

func Decode(s string) (*JsonObject, error) {
	return DecodeReader(strings.NewReader(s))
}

func DecodeBytes(b []byte) (*JsonObject, error) {
	return DecodeReader(bytes.NewReader(b))
}

func DecodeFile(fn string) (*JsonObject, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return DecodeReader(f)
}

func DecodeReader(reader io.Reader) (*JsonObject, error) {
	mm := map[string]interface{}{}
	err := json.NewDecoder(reader).Decode(&mm)
	if err != nil {
		return nil, err
	}

	return jsonObjectFromMap(&mm), nil
}

func jsonObjectFromMap(m *map[string]interface{}) *JsonObject {
	jsonObject := NewJsonObject()

	for key, val := range *m {
		switch v := val.(type) {
		case map[string]interface{}:
			(*jsonObject)[key] = jsonObjectFromMap(&v)
		case []interface{}:
			(*jsonObject)[key] = jsonArrayFromArray(&v)
		default:
			(*jsonObject)[key] = val
		}
	}

	return jsonObject
}

func jsonArrayFromArray(a *[]interface{}) *JsonArray {
	jsonArray := NewJsonArray(len(*a))
	for i, element := range *a {
		switch v := element.(type) {
		case map[string]interface{}:
			jsonArray.array[i] = jsonObjectFromMap(&v)
		case []interface{}:
			jsonArray.array[i] = jsonArrayFromArray(&v)
		default:
			jsonArray.array[i] = element
		}
	}
	return jsonArray
}

func toJsonType(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)

	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)

	case float32:
		return float64(v)
	}
	return val
}

func NewJsonObject() *JsonObject {
	return &JsonObject{}
}

func (j *JsonObject) Encode() (string, error) {
	b, err := j.EncodeBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (j *JsonObject) EncodeBytes() ([]byte, error) {
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (j *JsonObject) EncodeWriter(w io.Writer) (int, error) {
	b, err := j.EncodeBytes()
	if err != nil {
		return -1, err
	}
	return w.Write(b)
}

func (j *JsonObject) EncodePretty(indent string) (string, error) {
	b, err := j.EncodeBytesPretty(indent)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (j *JsonObject) EncodeBytesPretty(indent string) ([]byte, error) {
	b, err := json.MarshalIndent(j, "", indent)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (j *JsonObject) EncodeWriterPretty(indent string, w io.Writer) (int, error) {
	b, err := j.EncodeBytesPretty(indent)
	if err != nil {
		return -1, err
	}
	return w.Write(b)
}

func (j *JsonObject) Has(key string) bool {
	_, ok := (*j)[key]
	return ok
}

func (j *JsonObject) Put(key string, val interface{}) *JsonObject {
	(*j)[key] = toJsonType(val)
	return j
}

func (j *JsonObject) Delete(key string) *JsonObject {
	delete(*j, key)
	return j
}

func (j *JsonObject) GetObjectUnsafe(key string) *JsonObject {
	return (*j)[key].(*JsonObject)
}

func (j *JsonObject) GetObject(key string, def *JsonObject) *JsonObject {
	if val, ok := (*j)[key].(*JsonObject); ok {
		return val
	}
	return def
}

func (j *JsonObject) GetArrayUnsafe(key string) *JsonArray {
	return (*j)[key].(*JsonArray)
}

func (j *JsonObject) GetArray(key string, def *JsonArray) *JsonArray {
	if val, ok := (*j)[key].(*JsonArray); ok {
		return val
	}
	return def
}

func (j *JsonObject) GetStringUnsafe(key string) string {
	return (*j)[key].(string)
}

func (j *JsonObject) GetString(key string, def string) string {
	if val, ok := (*j)[key].(string); ok {
		return val
	}
	return def
}

func (j *JsonObject) GetIntUnsafe(key string) int {
	return int((*j)[key].(float64))
}

func (j *JsonObject) GetInt(key string, def int) int {
	if val, ok := (*j)[key].(float64); ok {
		return int(val)
	}
	return def
}

func (j *JsonObject) GetBoolUnsafe(key string) bool {
	return (*j)[key].(bool)
}

func (j *JsonObject) GetBool(key string, def bool) bool {
	if val, ok := (*j)[key].(bool); ok {
		return val
	}
	return def
}

func NewJsonArray(len int) *JsonArray {
	a := &JsonArray{}
	a.array = make([]interface{}, len)
	return a
}

func JsonArrayFromAny(elements ...any) *JsonArray {
	a := NewJsonArray(len(elements))
	for i, e := range elements {
		a.Set(i, e)
	}
	return a
}

func JsonArrayFromStrings(elements ...string) *JsonArray {
	a := NewJsonArray(len(elements))
	for i, e := range elements {
		a.array[i] = e
	}
	return a
}

func (a *JsonArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.array)
}

func (a *JsonArray) Len() int {
	return len(a.array)
}

func (a *JsonArray) Clear() *JsonArray {
	a.array = make([]interface{}, 0)
	return a
}

func (a *JsonArray) Add(val interface{}) *JsonArray {
	a.array = append(a.array, toJsonType(val))
	return a
}

func (a *JsonArray) Set(index int, val interface{}) *JsonArray {
	a.array[index] = toJsonType(val)
	return a
}

func (a *JsonArray) GetObjectUnsafe(index int) *JsonObject {
	return a.array[index].(*JsonObject)
}

func (a *JsonArray) GetObject(index int, def *JsonObject) *JsonObject {
	if index < 0 || index >= len(a.array) {
		return def
	}
	if val, ok := a.array[index].(*JsonObject); ok {
		return val
	}
	return def
}

func (a *JsonArray) GetArrayUnsafe(index int) *JsonArray {
	return a.array[index].(*JsonArray)
}

func (a *JsonArray) GetArray(index int, def *JsonArray) *JsonArray {
	if index < 0 || index >= len(a.array) {
		return def
	}
	if val, ok := a.array[index].(*JsonArray); ok {
		return val
	}
	return def
}

func (a *JsonArray) GetStringUnsafe(index int) string {
	return a.array[index].(string)
}

func (a *JsonArray) GetString(index int, def string) string {
	if index < 0 || index >= len(a.array) {
		return def
	}
	if val, ok := a.array[index].(string); ok {
		return val
	}
	return def
}

func (a *JsonArray) GetIntUnsafe(index int) int {
	return int(a.array[index].(float64))
}

func (a *JsonArray) GetInt(index int, def int) int {
	if index < 0 || index >= len(a.array) {
		return def
	}
	if val, ok := a.array[index].(float64); ok {
		return int(val)
	}
	return def
}

func (a *JsonArray) GetBoolUnsafe(index int) bool {
	return a.array[index].(bool)
}

func (a *JsonArray) GetBool(index int, def bool) bool {
	if index < 0 || index >= len(a.array) {
		return def
	}
	if val, ok := a.array[index].(bool); ok {
		return val
	}
	return def
}

func (a *JsonArray) ToAnyArray() []any {
	l := a.Len()
	ret := make([]any, l)
	for i := 0; i < l; i++ {
		ret[i] = a.array[i]
	}
	return ret
}

func (a *JsonArray) ToStringArrayUnsafe() []string {
	l := a.Len()
	ret := make([]string, l)
	for i := 0; i < l; i++ {
		ret[i] = a.GetStringUnsafe(i)
	}
	return ret
}

func (a *JsonArray) ToStringArray(def string) []string {
	l := a.Len()
	ret := make([]string, l)
	for i := 0; i < l; i++ {
		ret[i] = a.GetString(i, def)
	}
	return ret
}
