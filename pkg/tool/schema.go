package tool

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Reflect builds a JSON Schema for a tool's arguments object from a Go value's
// type. Pass a zero value of the parameter struct:
//
//	type Args struct {
//	    City  string   `json:"city"            desc:"City name, e.g. Paris"`
//	    Units string   `json:"units,omitempty" desc:"temperature unit" enum:"c,f"`
//	    Days  []string `json:"days,omitempty"`
//	}
//	schema := tool.Reflect(Args{})
//
// It is a small, dependency-free reflector covering the shapes tool arguments
// actually take: structs, scalars (string/bool/int*/uint*/float*), slices and
// arrays, maps, pointers, and embedded structs (flattened). A pointer field or
// an omitempty tag marks a field optional; everything else lands in "required".
// Fields tagged json:"-" or unexported are skipped. Struct tags read:
//
//	json:"name,omitempty"  field name + optionality
//	desc:"..."             property description
//	enum:"a,b,c"           allowed string values
//
// For schema shapes it does not express, write the json.RawMessage by hand and
// pass it to Func directly — Reflect is a convenience, not a requirement.
func Reflect(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	return marshalSchema(reflect.TypeOf(v))
}

// ReflectType is Reflect for a type known only at compile time:
//
//	schema := tool.ReflectType[Args]()
func ReflectType[T any]() json.RawMessage {
	var zero T
	return marshalSchema(reflect.TypeOf(&zero).Elem())
}

func marshalSchema(t reflect.Type) json.RawMessage {
	b, err := json.Marshal(schemaFor(t, map[reflect.Type]bool{}))
	if err != nil {
		return nil
	}
	return b
}

func schemaFor(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	t = deref(t)
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaFor(t.Elem(), seen)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(t.Elem(), seen)}
	case reflect.Struct:
		return structSchema(t, seen)
	default:
		// interface{}, chan, func, ... -> unconstrained
		return map[string]any{}
	}
}

func structSchema(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	if seen[t] {
		return map[string]any{"type": "object"} // break recursive types
	}
	seen[t] = true
	defer delete(seen, t)

	props := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, omitempty := parseJSONTag(f)
		if name == "-" {
			continue
		}
		// Embedded struct without an explicit json name: flatten its fields.
		if f.Anonymous && name == "" {
			if et := deref(f.Type); et.Kind() == reflect.Struct {
				sub := structSchema(et, seen)
				mergeObject(props, &required, sub)
				continue
			}
		}
		if name == "" {
			name = f.Name
		}

		ps := schemaFor(f.Type, seen)
		if d := f.Tag.Get("desc"); d != "" {
			ps["description"] = d
		}
		if e := f.Tag.Get("enum"); e != "" {
			if vals := splitList(e); len(vals) > 0 {
				ps["enum"] = vals
			}
		}
		props[name] = ps

		if !omitempty && f.Type.Kind() != reflect.Pointer {
			required = append(required, name)
		}
	}

	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func parseJSONTag(f reflect.StructField) (name string, omitempty bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			omitempty = true
		}
	}
	return parts[0], omitempty
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func mergeObject(props map[string]any, required *[]string, sub map[string]any) {
	if sp, ok := sub["properties"].(map[string]any); ok {
		for k, v := range sp {
			props[k] = v
		}
	}
	if rq, ok := sub["required"].([]string); ok {
		*required = append(*required, rq...)
	}
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
