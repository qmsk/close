package util

import (
	"testing"
)

type subStruct struct {
	Value string
}

type testStruct struct {
	CamelCaseDefault string
	NilDefault       *int
	NilTagged        *int    `display:"No data"`
	DisplayTagged    string  `display:"Extra name"`
	Slice            []string
	SliceStruct      []*subStruct
	unexported       string
	Map              map[string]string
}

func TestPrettySprintf(t *testing.T) {
	testVar := testStruct{
		CamelCaseDefault: "camel value",
		DisplayTagged: "extra value",
		Slice: []string{"one", "two"},
		SliceStruct: []*subStruct{ &subStruct{ Value: "value", } },
		unexported: "this should not show",
		Map: map[string]string{ "key1": "value1", "key2": "value2" },
	}

	testPoint := &testStruct {
		CamelCaseDefault: "camel value in a pointer",
		DisplayTagged: "extra value in a pointer",
	}
	t.Log(PrettySprintf("testVar", testVar))
	t.Log(PrettySprintf("testPointer", testPoint))
}
