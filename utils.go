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
