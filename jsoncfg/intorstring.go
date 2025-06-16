package jsoncfg

import (
	"encoding/json"
	"unsafe"
)

// IntOrStringKind represents the kind of the [IntOrString] value.
type IntOrStringKind int

const (
	IntOrStringKindInvalid IntOrStringKind = iota
	IntOrStringKindInt
	IntOrStringKindString
)

// IntOrString is a JSON value that can be either an int or a string.
type IntOrString struct {
	kind IntOrStringKind
	data *byte
	len  int // also used as storage for int
}

// IntOrStringFromInt returns i as an [IntOrString] value.
func IntOrStringFromInt(i int) IntOrString {
	return IntOrString{
		kind: IntOrStringKindInt,
		len:  i,
	}
}

// IntOrStringFromString returns s as an [IntOrString] value.
func IntOrStringFromString(s string) IntOrString {
	return IntOrString{
		kind: IntOrStringKindString,
		data: unsafe.StringData(s),
		len:  len(s),
	}
}

// Kind returns the value kind.
func (v IntOrString) Kind() IntOrStringKind {
	return v.kind
}

// IsValid returns whether the value is valid.
func (v IntOrString) IsValid() bool {
	return v.kind != IntOrStringKindInvalid
}

// IsInt returns whether the value is an int.
func (v IntOrString) IsInt() bool {
	return v.kind == IntOrStringKindInt
}

// IsString returns whether the value is a string.
func (v IntOrString) IsString() bool {
	return v.kind == IntOrStringKindString
}

// Int returns the int value.
// It panics if the value is not an int.
func (v IntOrString) Int() int {
	if v.kind != IntOrStringKindInt {
		panic("IntOrString: not an int")
	}
	return v.len
}

func (v IntOrString) string() string {
	return unsafe.String(v.data, v.len)
}

// String returns the string value.
// It panics if the value is not a string.
func (v IntOrString) String() string {
	if v.kind != IntOrStringKindString {
		panic("IntOrString: not a string")
	}
	return v.string()
}

// Equals returns whether the value is equal to other.
func (v IntOrString) Equals(other IntOrString) bool {
	if v.kind != other.kind {
		return false
	}
	switch v.kind {
	case IntOrStringKindInt:
		return v.len == other.len
	case IntOrStringKindString:
		return v.string() == other.string()
	default:
		return true
	}
}

// MarshalJSON implements [json.Marshaler].
func (v IntOrString) MarshalJSON() ([]byte, error) {
	switch v.kind {
	case IntOrStringKindInt:
		return json.Marshal(v.len)
	case IntOrStringKindString:
		return json.Marshal(v.string())
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON implements [json.Unmarshaler].
func (v *IntOrString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = IntOrString{}
		return nil
	}

	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*v = IntOrStringFromString(s)
		return nil
	}

	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	*v = IntOrStringFromInt(i)
	return nil
}
