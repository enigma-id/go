package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/enigma-id/go/validation"
)

type (
	// Binder is the interface that wraps the Bind method.
	Binder interface {
		Bind(i interface{}, c *Context) error
	}

	// DefaultBinder is the default implementation of the Binder interface.
	DefaultBinder struct{}

	// BindUnmarshaler is the interface used to wrap the UnmarshalParam method.
	BindUnmarshaler interface {
		// UnmarshalParam decodes and assigns a value from an form or query param.
		UnmarshalParam(param string) error
	}

	// binderValidator validation object.
	binderValidator struct {
		once      sync.Once
		validator *validation.Validator
	}
)

// Bind implements the `Binder#Bind` function.
func (b *DefaultBinder) Bind(i interface{}, c *Context) (err error) {
	req := c.Request()
	if req.ContentLength == 0 {
		if req.Method == http.MethodGet || req.Method == http.MethodDelete {
			if err = b.bindData(i, c.QueryParams(), "query"); err != nil {
				err = NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}

			if req.Method == http.MethodDelete {
				err = c.validator.Validate(i)
			}

		} else {
			err = NewHTTPError(http.StatusBadRequest, "Request body can't be empty")
		}
	} else {
		ctype := req.Header.Get(HeaderContentType)

		if strings.HasPrefix(ctype, MIMEApplicationJSON) {
			if err = json.NewDecoder(req.Body).Decode(i); err != nil {
				if _, ok := err.(*json.UnmarshalTypeError); ok {
					err = NewHTTPError(http.StatusBadRequest, "Incorrect data structure")
				} else if _, ok := err.(*json.SyntaxError); ok {
					err = NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
				}

				err = NewHTTPError(http.StatusBadRequest, err.Error())
			} else {
				err = c.validator.Validate(i)
			}
		} else {
			err = ErrUnsupportedMediaType
		}
	}

	return
}

func (b *DefaultBinder) bindData(ptr interface{}, data map[string][]string, tag string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If tag is nil, we inspect if the field is a struct.
			if _, ok := bindUnmarshaler(structField); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
				continue
			}
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			inputFieldName = strings.ToLower(inputFieldName)
			for k, v := range data {
				if strings.ToLower(k) == inputFieldName {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// Call this first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return err

		}
	}
	return nil
}

// Validate the request when binding
func (v *binderValidator) Validate(obj interface{}) (err error) {
	v.lazyinit()

	var o *validation.Response
	if vr, ok := obj.(validation.Request); ok {
		o = v.validator.Request(vr)
	} else {
		o = v.validator.Struct(obj)
	}

	if !o.Valid {
		err = o
	}

	return
}

// lazyinit initialing validator instances for one of time only.
func (v *binderValidator) lazyinit() {
	v.once.Do(func() {
		v.validator = validation.New()
	})
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	// But also call it here, in case we're dealing with an array of BindUnmarshalers
	if ok, err := unmarshalField(valueKind, val, structField); ok {
		return err
	}

	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func unmarshalField(valueKind reflect.Kind, val string, field reflect.Value) (bool, error) {
	switch valueKind {
	case reflect.Ptr:
		return unmarshalFieldPtr(val, field)
	default:
		return unmarshalFieldNonPtr(val, field)
	}
}

func bindUnmarshaler(field reflect.Value) (BindUnmarshaler, bool) {
	ptr := reflect.New(field.Type())
	if ptr.CanInterface() {
		iface := ptr.Interface()
		if unmarshaler, ok := iface.(BindUnmarshaler); ok {
			return unmarshaler, ok
		}
	}
	return nil, false
}

func unmarshalFieldNonPtr(value string, field reflect.Value) (bool, error) {
	if unmarshaler, ok := bindUnmarshaler(field); ok {
		err := unmarshaler.UnmarshalParam(value)
		field.Set(reflect.ValueOf(unmarshaler).Elem())
		return true, err
	}
	return false, nil
}

func unmarshalFieldPtr(value string, field reflect.Value) (bool, error) {
	if field.IsNil() {
		// Initialize the pointer to a nil value
		field.Set(reflect.New(field.Type().Elem()))
	}
	return unmarshalFieldNonPtr(value, field.Elem())
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
