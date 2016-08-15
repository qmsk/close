package util

import (
	"testing"
)

type SubStruct struct {
	Value string
}

type TestStruct struct {
	CamelCaseDefault string
	NilDefault       *int
	NilTagged        *int    `display:"No data"`
	DisplayTagged    string  `display:"Extra name"`
	Slice            []string
	SliceStruct      []*SubStruct
}

func TestPrettySprintf(t *testing.T) {
	testVar := TestStruct{
		CamelCaseDefault: "camel value",
		DisplayTagged: "extra value",
		Slice: []string{"one", "two"},
		SliceStruct: []*SubStruct{ &SubStruct{ Value: "value", } },
	}

	testPoint := &TestStruct {
		CamelCaseDefault: "camel value in a pointer",
		DisplayTagged: "extra value in a pointer",
	}
	t.Log(PrettySprintf("testVar", testVar))
	t.Log(PrettySprintf("testPointer", testPoint))
}
