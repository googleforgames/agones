package jsonpatch

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

var point = `{"type":"Point", "coordinates":[0.0, 1.0]}`
var lineString = `{"type":"LineString", "coordinates":[[0.0, 1.0], [2.0, 3.0]]}`

func TestPointLineStringReplace(t *testing.T) {
	patch, e := CreatePatch([]byte(point), []byte(lineString))
	assert.NoError(t, e)
	assert.Equal(t, len(patch), 3, "they should be equal")
	sort.Sort(ByPath(patch))
	change := patch[0]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/coordinates/0", "they should be equal")
	assert.Equal(t, change.Value, []interface{}{0.0, 1.0}, "they should be equal")
	change = patch[1]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/coordinates/1", "they should be equal")
	assert.Equal(t, change.Value, []interface{}{2.0, 3.0}, "they should be equal")
	change = patch[2]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/type", "they should be equal")
	assert.Equal(t, change.Value, "LineString", "they should be equal")
}

func TestLineStringPointReplace(t *testing.T) {
	patch, e := CreatePatch([]byte(lineString), []byte(point))
	assert.NoError(t, e)
	assert.Equal(t, len(patch), 3, "they should be equal")
	sort.Sort(ByPath(patch))
	change := patch[0]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/coordinates/0", "they should be equal")
	assert.Equal(t, change.Value, 0.0, "they should be equal")
	change = patch[1]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/coordinates/1", "they should be equal")
	assert.Equal(t, change.Value, 1.0, "they should be equal")
	change = patch[2]
	assert.Equal(t, change.Operation, "replace", "they should be equal")
	assert.Equal(t, change.Path, "/type", "they should be equal")
	assert.Equal(t, change.Value, "Point", "they should be equal")
}
