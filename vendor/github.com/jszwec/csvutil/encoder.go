package csvutil

import (
	"bytes"
	"reflect"
)

type encField struct {
	field
	encodeFunc
}

type encCache struct {
	typeKey typeKey
	buf     bytes.Buffer
	cache   []encField
	index   []int
	record  []string
}

func (c *encCache) fields(k typeKey) ([]encField, error) {
	if c.typeKey != k {
		fields := cachedFields(k)
		encFields := make([]encField, len(fields))

		for i, f := range fields {
			fn, err := encodeFn(f.typ)
			if err != nil {
				return nil, err
			}

			encFields[i] = encField{
				field:      f,
				encodeFunc: fn,
			}
		}
		c.cache, c.typeKey = encFields, k
	}
	return c.cache, nil
}

func (c *encCache) reset(fieldsLen int) {
	c.buf.Reset()

	if fieldsLen != len(c.index) {
		c.index = make([]int, fieldsLen)
		c.record = make([]string, fieldsLen)
		return
	}

	for i := range c.index {
		c.index[i] = 0
		c.record[i] = ""
	}
}

// Encoder writes structs CSV representations to the output stream.
type Encoder struct {
	// Tag defines which key in the struct field's tag to scan for names and
	// options (Default: 'csv').
	Tag string

	// If AutoHeader is true, a struct header is encoded during the first call
	// to Encode automatically (Default: true).
	AutoHeader bool

	w        Writer
	cache    encCache
	noHeader bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w Writer) *Encoder {
	return &Encoder{
		w:          w,
		noHeader:   true,
		AutoHeader: true,
	}
}

// Encode writes the CSV encoding of v to the output stream. The provided
// argument v must be a non-nil struct.
//
// Only the exported fields will be encoded.
//
// First call to Encode will write a header unless EncodeHeader was called first
// or AutoHeader is false. Header names can be customized by using tags
// ('csv' by default), otherwise original Field names are used.
//
// Header and fields are written in the same order as struct fields are defined.
// Embedded struct's fields are treated as if they were part of the outer struct.
// Fields that are embedded types and that are tagged are treated like any
// other field, but they have to implement Marshaler or encoding.TextMarshaler
// interfaces.
//
// Marshaler interface has the priority over encoding.TextMarshaler.
//
// Tagged fields have the priority over non tagged fields with the same name.
//
// Following the Go visibility rules if there are multiple fields with the same
// name (tagged or not tagged) on the same level and choice between them is
// ambiguous, then all these fields will be ignored.
//
// Nil values will be encoded as empty strings. Same will happen if 'omitempty'
// tag is set, and the value is a default value like 0, false or nil interface.
//
// Bool types are encoded as 'true' or 'false'.
//
// Float types are encoded using strconv.FormatFloat with precision -1 and 'G'
// format. NaN values are encoded as 'NaN' string.
//
// Fields of type []byte are being encoded as base64-encoded strings.
//
// Fields can be excluded from encoding by using '-' tag option.
//
// Examples of struct tags:
//
// 	// Field appears as 'myName' header in CSV encoding.
// 	Field int `csv:"myName"`
//
// 	// Field appears as 'Field' header in CSV encoding.
// 	Field int
//
// 	// Field appears as 'myName' header in CSV encoding and is an empty string
//	// if Field is 0.
// 	Field int `csv:"myName,omitempty"`
//
// 	// Field appears as 'Field' header in CSV encoding and is an empty string
//	// if Field is 0.
// 	Field int `csv:",omitempty"`
//
// 	// Encode ignores this field.
// 	Field int `csv:"-"`
//
// Encode doesn't flush data. The caller is responsible for calling Flush() if
// the used Writer supports it.
func (e *Encoder) Encode(v interface{}) error {
	return e.encode(reflect.ValueOf(v))
}

// EncodeHeader writes the CSV header of the provided struct value to the output
// stream. The provided argument v must be a struct value.
//
// The first Encode method call will not write header if EncodeHeader was called
// before it. This method can be called in cases when a data set could be
// empty, but header is desired.
//
// EncodeHeader is like Header function, but it works with the Encoder and writes
// directly to the output stream. Look at Header documentation for the exact
// header encoding rules.
func (e *Encoder) EncodeHeader(v interface{}) error {
	val := reflect.ValueOf(v)
	if !val.IsValid() {
		return &UnsupportedTypeError{}
	}

	typ := walkType(val.Type())
	if typ.Kind() != reflect.Struct {
		return &UnsupportedTypeError{Type: typ}
	}

	return e.encodeHeader(typ)
}

func (e *Encoder) encode(v reflect.Value) error {
	v = walkValue(v)

	if !v.IsValid() {
		return &InvalidEncodeError{}
	}

	if v.Kind() != reflect.Struct {
		return &InvalidEncodeError{v.Type()}
	}

	if e.AutoHeader && e.noHeader {
		if err := e.encodeHeader(v.Type()); err != nil {
			return err
		}
	}

	return e.marshal(v)
}

func (e *Encoder) encodeHeader(typ reflect.Type) (err error) {
	defer func() {
		if err == nil {
			e.noHeader = false
		}
	}()

	fields, err := e.fields(typ)
	if err != nil {
		return err
	}

	e.cache.reset(len(fields))
	for i, f := range fields {
		e.cache.record[i] = f.tag.name
	}
	return e.w.Write(e.cache.record)
}

func (e *Encoder) marshal(v reflect.Value) error {
	fields, err := e.fields(v.Type())
	if err != nil {
		return err
	}

	e.cache.reset(len(fields))
	buf, index, record := &e.cache.buf, e.cache.index, e.cache.record

	for i, f := range fields {
		v := walkIndex(v, f.index)
		if !v.IsValid() {
			continue
		}

		n, err := f.encodeFunc(v, buf, f.tag.omitEmpty)
		if err != nil {
			return err
		}
		index[i] = n
	}

	out := buf.String()
	for i, n := range index {
		record[i], out = out[:n], out[n:]
	}

	return e.w.Write(record)
}

func (e *Encoder) tag() string {
	if e.Tag == "" {
		return defaultTag
	}
	return e.Tag
}

func (e *Encoder) fields(typ reflect.Type) ([]encField, error) {
	return e.cache.fields(typeKey{e.tag(), typ})
}

func walkIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		v = walkPtr(v)
		if !v.IsValid() {
			return reflect.Value{}
		}
		v = v.Field(i)
	}

	return walkPtr(v)
}

func walkPtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func walkValue(v reflect.Value) reflect.Value {
	for {
		switch v.Kind() {
		case reflect.Ptr, reflect.Interface:
			v = v.Elem()
		default:
			return v
		}
	}
}

func walkType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}
