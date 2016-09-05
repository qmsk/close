package util

import (
	"testing"
)

type idStruct struct {
	Value string `positional-arg-name:"id"`
}

type configStruct struct {
	Sub idStruct `positional-args:"id-struct"`
}

func TestExpandPath(t *testing.T) {
	res, err := ExpandPath("/:id", &configStruct{ idStruct{"value"} })
	if res != "/value" {
		t.Errorf("Result is not what expected (/value): %v", res)
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}
}
