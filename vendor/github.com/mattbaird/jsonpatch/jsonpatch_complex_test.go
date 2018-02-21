package jsonpatch

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

var complexBase = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}}`
var complexA = `{"a":100, "b":[{"c1":"goodbye", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}}`
var complexB = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":100, "g":"h", "i":"j"}}`
var complexC = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}, "k":[{"l":"m"}, {"l":"o"}]}`
var complexD = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"}, {"c3":"hello3", "d3":"foo3"} ], "e":{"f":200, "g":"h", "i":"j"}}`
var complexE = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}}`

func TestComplexSame(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(complexBase))
	assert.NoError(t, e)
	assert.Equal(t, 0, len(patch), "they should be equal")
}
func TestComplexOneStringReplaceInArray(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(complexA))
	assert.NoError(t, e)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/b/0/c1", change.Path, "they should be equal")
	assert.Equal(t, "goodbye", change.Value, "they should be equal")
}

func TestComplexOneIntReplace(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(complexB))
	assert.NoError(t, e)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "replace", change.Operation, "they should be equal")
	assert.Equal(t, "/e/f", change.Path, "they should be equal")
	var expected float64 = 100
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestComplexOneAdd(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(complexC))
	assert.NoError(t, e)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/k", change.Path, "they should be equal")
	a := make(map[string]interface{})
	b := make(map[string]interface{})
	a["l"] = "m"
	b["l"] = "o"
	expected := []interface{}{a, b}
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestComplexOneAddToArray(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(complexC))
	assert.NoError(t, e)
	assert.Equal(t, 1, len(patch), "they should be equal")
	change := patch[0]
	assert.Equal(t, "add", change.Operation, "they should be equal")
	assert.Equal(t, "/k", change.Path, "they should be equal")
	a := make(map[string]interface{})
	b := make(map[string]interface{})
	a["l"] = "m"
	b["l"] = "o"
	expected := []interface{}{a, b}
	assert.Equal(t, expected, change.Value, "they should be equal")
}

func TestComplexVsEmpty(t *testing.T) {
	patch, e := CreatePatch([]byte(complexBase), []byte(empty))
	assert.NoError(t, e)
	assert.Equal(t, 3, len(patch), "they should be equal")
	sort.Sort(ByPath(patch))
	change := patch[0]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/a", change.Path, "they should be equal")

	change = patch[1]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/b", change.Path, "they should be equal")

	change = patch[2]
	assert.Equal(t, "remove", change.Operation, "they should be equal")
	assert.Equal(t, "/e", change.Path, "they should be equal")
}
