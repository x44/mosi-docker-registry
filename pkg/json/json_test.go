package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimple(t *testing.T) {
	assert := assert.New(t)

	json := NewJsonObject()

	json.Put("key_string_a", "stringa").Put("key_string_b", "stringb")
	json.Put("key_int_a", 123).Put("key_int_b", 987)
	json.Put("key_bool_a", true).Put("key_bool_b", false)

	assert.False(json.Has("not_exists"))
	assert.True(json.Has("key_string_a"))
	assert.True(json.Has("key_int_a"))
	assert.Equal("stringa", json.GetString("key_string_a", ""))
	assert.Equal("default", json.GetString("not_exists", "default"))
	assert.Equal(123, json.GetInt("key_int_a", -1))
	assert.Equal(true, json.GetBool("key_bool_a", false))

	json.Delete("key_string_a")
	assert.False(json.Has("key_string_a"))
}

func TestObjects(t *testing.T) {
	assert := assert.New(t)

	json := NewJsonObject()
	child := NewJsonObject()
	childchild := NewJsonObject()

	json.Put("child", child)
	child.Put("childchild", childchild)

	child.Put("key_string_a", "stringa")
	childchild.Put("key_string_b", "stringb")

	assert.True(json.Has("child"))
	assert.Equal("stringa", json.GetObjectUnsafe("child").GetString("key_string_a", ""))
	assert.Equal("stringb", json.GetObjectUnsafe("child").GetObjectUnsafe("childchild").GetString("key_string_b", ""))
}

func TestArrays(t *testing.T) {
	assert := assert.New(t)

	json := NewJsonObject()
	array := NewJsonArray(0)
	arrayarray := NewJsonArray(4)
	child := NewJsonObject()
	child.Put("key", "val")

	json.Put("array", array)

	array.Add("element0")
	array.Add(1)
	array.Add(true)
	array.Add(arrayarray)

	assert.Equal(4, json.GetArrayUnsafe("array").Len())
	assert.Equal("element0", json.GetArrayUnsafe("array").GetStringUnsafe(0))
	assert.Equal(1, json.GetArrayUnsafe("array").GetIntUnsafe(1))
	assert.Equal(true, json.GetArrayUnsafe("array").GetBoolUnsafe(2))

	assert.Equal("empty", json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetString(0, "empty"))
	assert.Equal(-1, json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetInt(1, -1))
	assert.Equal(false, json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetBool(2, false))
	assert.Nil(json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetObject(3, nil))
	assert.Nil(json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetObject(10, nil))

	arrayarray.Set(0, "element0")
	arrayarray.Set(1, 1)
	arrayarray.Set(2, true)
	arrayarray.Set(3, child)

	assert.Equal(4, json.GetArrayUnsafe("array").GetArrayUnsafe(3).Len())
	assert.Equal("element0", json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetStringUnsafe(0))
	assert.Equal(1, json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetIntUnsafe(1))
	assert.Equal(true, json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetBoolUnsafe(2))
	assert.Equal("val", json.GetArrayUnsafe("array").GetArrayUnsafe(3).GetObjectUnsafe(3).GetStringUnsafe("key"))
}

func TestEncDec(t *testing.T) {
	assert := assert.New(t)

	json1 := NewJsonObject()
	sub1 := NewJsonObject()
	array := NewJsonArray(0)
	arrayarray := NewJsonArray(4)
	child1 := NewJsonObject()
	child1.Put("key", "val")

	json1.Put("sub", sub1)
	sub1.Put("subkey", "subval")

	json1.Put("array", array)

	json1.Put("key_string_a", "stringa").Put("key_string_b", "stringb")
	json1.Put("key_int_a", 123).Put("key_int_b", 987)
	json1.Put("key_bool_a", true).Put("key_bool_b", false)

	array.Add("element0")
	array.Add(1)
	array.Add(true)
	array.Add(arrayarray)

	arrayarray.Set(0, "element0")
	arrayarray.Set(1, 1)
	arrayarray.Set(2, true)
	arrayarray.Set(3, child1)

	s, err := json1.EncodePretty("\t")
	assert.Nil(err, "Encode failed %v", err)

	json2, err := Decode(s)
	assert.Nil(err, "Decode failed %v", err)

	sub2 := json2.GetObjectUnsafe("sub")

	assert.Equal(json1.GetStringUnsafe("key_string_a"), json2.GetStringUnsafe("key_string_a"))
	assert.Equal(json1.GetIntUnsafe("key_int_a"), json2.GetIntUnsafe("key_int_a"))
	assert.Equal(sub1.GetStringUnsafe("subkey"), sub2.GetStringUnsafe("subkey"))

	assert.Equal(4, json2.GetArrayUnsafe("array").GetArrayUnsafe(3).Len())
	assert.Equal("element0", json2.GetArrayUnsafe("array").GetArrayUnsafe(3).GetStringUnsafe(0))
	assert.Equal(1, json2.GetArrayUnsafe("array").GetArrayUnsafe(3).GetIntUnsafe(1))
	assert.Equal(true, json2.GetArrayUnsafe("array").GetArrayUnsafe(3).GetBoolUnsafe(2))
	assert.Equal("val", json2.GetArrayUnsafe("array").GetArrayUnsafe(3).GetObjectUnsafe(3).GetStringUnsafe("key"))
}
