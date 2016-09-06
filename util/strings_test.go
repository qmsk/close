package util

import (
	"testing"
)

type idStruct struct {
	Id string `positional-arg-name:"id"`
	Type string `positional-arg-name:"type"`
}

type configStruct struct {
	Sub idStruct `positional-args:"id-struct"`
}

func TestExpandPath(t *testing.T) {
	res, err := ExpandPath("/:id/:type", &configStruct{ idStruct{"123", "ping"} })
	if res != "/123/ping" {
		t.Errorf("Result is not what expected (/123/ping): %v", res)
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}
}
