package jsonpatch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMarshalNullableValue(t *testing.T) {
	p1 := JsonPatchOperation{
		Operation: "replace",
		Path:      "/a1",
		Value:     nil,
	}
	assert.JSONEq(t, `{"op":"replace", "path":"/a1","value":null}`, p1.Json())

	p2 := JsonPatchOperation{
		Operation: "replace",
		Path:      "/a2",
		Value:     "v2",
	}
	assert.JSONEq(t, `{"op":"replace", "path":"/a2", "value":"v2"}`, p2.Json())
}

func TestMarshalNonNullableValue(t *testing.T) {
	p1 := JsonPatchOperation{
		Operation: "remove",
		Path:      "/a1",
	}
	assert.JSONEq(t, `{"op":"remove", "path":"/a1"}`, p1.Json())

}
