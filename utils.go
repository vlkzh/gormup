package gormup

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/shockerli/cvt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func isSupportForUpdate(v any) bool {
	switch v.(type) {
	case sql.NamedArg,
		clause.Column,
		clause.Table,
		clause.Interface,
		clause.Expression,
		gorm.Valuer,
		*gorm.DB:
		return false
	}
	return true
}

func toString(v any) string {
	if dv, ok := v.(driver.Valuer); ok && dv != nil {
		if xv, err := dv.Value(); err == nil {
			v = xv
		}
	}

	if v == nil {
		return ""
	}
	rfv := reflect.ValueOf(v)
	if rfv.Kind() == reflect.Ptr && rfv.IsNil() {
		return ""
	}

	strVal, err := cvt.StringE(v)
	if err == nil {
		return strVal
	}

	byteVal, err := json.Marshal(v)
	if err == nil {
		return string(byteVal)
	}

	return ""
}

func getModelType(v reflect.Value) reflect.Type {
	t := v.Type()
	for {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			t = t.Elem()
		default:
			return t
		}
	}
}

func isValidStruct(v reflect.Value) bool {
	if isStruct(v.Type()) {
		return true
	}
	if !isPointerOfStruct(v.Type()) {
		return false
	}
	return !v.IsZero() && !v.IsNil() && v.IsValid()
}

func isArray(v reflect.Type) bool {
	return v.Kind() == reflect.Array ||
		v.Kind() == reflect.Slice
}

func isPointerOfArray(v reflect.Type) bool {
	return v.Kind() == reflect.Ptr && isArray(v.Elem())
}

func isStruct(v reflect.Type) bool {
	return v.Kind() == reflect.Struct
}

func isPointerOfStruct(v reflect.Type) bool {
	return v.Kind() == reflect.Ptr && isStruct(v.Elem())
}

func is2PointerOfStruct(v reflect.Type) bool {
	return v.Kind() == reflect.Ptr && isPointerOfStruct(v.Elem())
}

func setValue(target reflect.Value, values ...reflect.Value) {
	if target.Kind() == reflect.Interface {
		target = target.Elem()
	}

	if len(values) == 0 {
		return
	}

	if isPointerOfArray(target.Type()) {
		newVal := reflect.MakeSlice(target.Elem().Type(), len(values), len(values))
		for i, v := range values {
			el := newVal.Index(i).Interface()
			setValue(reflect.ValueOf(&el), v)
		}
		target.Elem().Set(newVal)
	} else {
		value := values[0]
		if value.Kind() == reflect.Interface {
			value = value.Elem()
		}

		if is2PointerOfStruct(target.Type()) {
			if isPointerOfStruct(value.Type()) {
				target.Elem().Set(value)
			} else if isStruct(value.Type()) {
				p := reflect.New(target.Type().Elem())
				p.Elem().Set(value)
				target.Elem().Set(p)
			}
		} else if isPointerOfStruct(target.Type()) {
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			if isPointerOfStruct(value.Type()) {
				target.Elem().Set(value.Elem())
			} else if isStruct(value.Type()) {
				target.Elem().Set(value)
			}
		}
	}
}
