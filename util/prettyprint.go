package util

import (
	"fmt"
	"reflect"
)

const (
	INDENT_WIDTH = 2
)

func PrettySprintf(name string, data interface{}) string {
	return prettySprintf(name, reflect.ValueOf(data), 0)
}

func prettySprintf(name string, v reflect.Value, level int) (output string) {
	var indent, indentFormat string

	if !v.CanInterface() {
		return
	}
	// output = indent + name + ": \n"

	// Contstruct the indentation string, first the format...
	indentFormat = fmt.Sprintf("%%%ds", INDENT_WIDTH * level)
	// ... then the string itself
	indent = fmt.Sprintf(indentFormat, "")

	namePrefix := ""
	if name != "" {
		namePrefix = name + ": "
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			t := v.Type().Field(i)
			fieldName := t.Tag.Get("display")
			if fieldName == "" {
				fieldName = t.Name
			}
			output = output + prettySprintf(fieldName, f, level+1)
		}
	case reflect.Slice:
		output = indent + namePrefix + "[\n"
		for j := 0; j < v.Len(); j++ {
			indexName := fmt.Sprintf("%d", j)
			output = output + prettySprintf(indexName, v.Index(j), level+1)
		}
		output = output + indent + "]\n"
	case reflect.Ptr:
		if v.IsNil() {
			// output = indent + fmt.Sprintf("%s is empty\n", name)
		} else {
			output = prettySprintf(name, v.Elem(), level)
		}
	case reflect.Map:
		output = indent + namePrefix + "[\n"
		for _, k := range v.MapKeys() {
			keyName := fmt.Sprintf("%v", k.Interface())
			output = output + prettySprintf(keyName, v.MapIndex(k), level+1)
		}
		output = output + indent + "]\n"
	default:
		output = indent + fmt.Sprintf("%s%v\n", namePrefix, v.Interface())
	}
	return
}
