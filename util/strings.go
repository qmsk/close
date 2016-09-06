package util

import (
	"fmt"
	"reflect"
	"strings"
)

func ExpandPath(path string, extra interface{}) (string, error) {
	components := strings.Split(path, "/")
	values := make(map[string]string)

	extraPtrToV := reflect.ValueOf(extra)
	v := extraPtrToV.Elem()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		t := v.Type().Field(i)
		fieldName := t.Tag.Get("positional-args")
		if fieldName == "" {
			continue
		}

		for j := 0; j < f.NumField(); j++ {
			arg := f.Field(j)
			argT := f.Type().Field(j)
			argName := argT.Tag.Get("positional-arg-name")
			if argName == "" {
				continue
			}

			values[argName] = arg.Interface().(string)
		}
	}

	for n, c := range components {
		if strings.HasPrefix(c, ":") {
			if val, found := values[c[1:]]; !found {
				return "", fmt.Errorf("Error expanding path with request parameters, %v not found in the given configuration structure", c)
			} else {
				components[n] = val
			}
		}
	}

	return strings.Join(components, "/"), nil
}
