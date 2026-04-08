package yaml

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func Marshal(v any) ([]byte, error) {
	return marshalValue(reflect.ValueOf(v), 0)
}

func marshalValue(v reflect.Value, indent int) ([]byte, error) {
	if !v.IsValid() {
		return []byte("null\n"), nil
	}

	t := v.Type()
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		if v.IsNil() {
			return []byte("null\n"), nil
		}

		return marshalValue(v.Elem(), indent)
	}

	switch t.Kind() {
	case reflect.String:
		return marshalString(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Appendf(nil, "%d\n", v.Int()), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Appendf(nil, "%g\n", v.Float()), nil
	case reflect.Bool:
		if v.Bool() {
			return []byte("true\n"), nil
		}

		return []byte("false\n"), nil
	case reflect.Slice, reflect.Array:
		return marshalSlice(v, indent)
	case reflect.Map:
		return marshalMap(v, indent)
	case reflect.Struct:
		return marshalStruct(v, indent)
	default:
		return marshalString(fmt.Sprintf("%v", v.Interface()))
	}
}

func marshalString(s string) ([]byte, error) {
	if s == "" {
		return []byte("\"\"\n"), nil
	}

	if needsQuoting(s) {
		quoted, err := json.Marshal(s)
		if err != nil {
			return nil, fmt.Errorf("yaml: marshal string: %w", err)
		}

		return append(quoted, '\n'), nil
	}

	return []byte(s + "\n"), nil
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	special := ":{}[],&*?|->!%@`#'\"\\\n\r\t "
	for _, c := range s {
		if strings.ContainsRune(special, c) {
			return true
		}
	}

	if s == "true" || s == "false" || s == "null" {
		return true
	}

	return false
}

func marshalSlice(v reflect.Value, indent int) ([]byte, error) {
	if v.Len() == 0 {
		return []byte("[]\n"), nil
	}

	prefix := strings.Repeat("  ", indent)

	var buf []byte

	for i := range v.Len() {
		elem, err := marshalValue(v.Index(i), indent+1)
		if err != nil {
			return nil, err
		}

		buf = append(buf, prefix+"- "...)
		lines := strings.Split(strings.TrimRight(string(elem), "\n"), "\n")

		buf = append(buf, lines[0]...)
		for _, line := range lines[1:] {
			buf = append(buf, '\n')
			buf = append(buf, prefix+"  "...)
			buf = append(buf, line...)
		}

		buf = append(buf, '\n')
	}

	return buf, nil
}

func marshalMap(v reflect.Value, indent int) ([]byte, error) {
	if v.Len() == 0 {
		return []byte("{}\n"), nil
	}

	prefix := strings.Repeat("  ", indent)

	var buf []byte

	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	for _, key := range keys {
		val := v.MapIndex(key)
		keyStr := key.String()

		valBytes, err := marshalValue(val, indent+1)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(strings.TrimRight(string(valBytes), "\n"), "\n")

		if needsQuoting(keyStr) {
			quoted, _ := json.Marshal(keyStr)
			buf = append(buf, prefix+string(quoted)+": "...)
		} else {
			buf = append(buf, prefix+keyStr+": "...)
		}

		buf = append(buf, lines[0]...)
		for _, line := range lines[1:] {
			buf = append(buf, '\n')
			buf = append(buf, prefix+"  "...)
			buf = append(buf, line...)
		}

		buf = append(buf, '\n')
	}

	return buf, nil
}

func marshalStruct(v reflect.Value, indent int) ([]byte, error) {
	t := v.Type()

	fields := make([]structField, 0, t.NumField())
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("yaml")
		if tag == "-" {
			continue
		}

		name := field.Name
		if tag != "" {
			name = strings.Split(tag, ",")[0]
		} else {
			if jt := field.Tag.Get("json"); jt != "" {
				parts := strings.Split(jt, ",")
				if parts[0] != "" && parts[0] != "-" {
					name = parts[0]
				}
			}
		}

		fields = append(fields, structField{name: name, value: v.Field(i)})
	}

	return marshalFields(fields, indent)
}

type structField struct {
	name  string
	value reflect.Value
}

func marshalFields(fields []structField, indent int) ([]byte, error) {
	prefix := strings.Repeat("  ", indent)

	var buf []byte

	for _, f := range fields {
		valBytes, err := marshalValue(f.value, indent+1)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(strings.TrimRight(string(valBytes), "\n"), "\n")

		if needsQuoting(f.name) {
			quoted, _ := json.Marshal(f.name)
			buf = append(buf, prefix+string(quoted)+": "...)
		} else {
			buf = append(buf, prefix+f.name+": "...)
		}

		buf = append(buf, lines[0]...)
		for _, line := range lines[1:] {
			buf = append(buf, '\n')
			buf = append(buf, prefix+"  "...)
			buf = append(buf, line...)
		}

		buf = append(buf, '\n')
	}

	return buf, nil
}
