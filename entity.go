package gormup

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm/schema"
)

type entity struct {
	schema       *schema.Schema
	reflectValue reflect.Value

	primaryKey       string
	otherPrimaryKeys []string

	id     string
	fields map[string]string
}

func createEntity(
	ctx context.Context,
	sch *schema.Schema,
	otherPrimaryKeys []string,
	v reflect.Value,
) *entity {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if !isValidStruct(v) {
		return nil
	}
	if len(sch.PrimaryFields) != 1 {
		return nil
	}

	e := &entity{
		schema:           sch,
		primaryKey:       sch.PrioritizedPrimaryField.DBName,
		otherPrimaryKeys: otherPrimaryKeys,
		reflectValue:     v,
	}

	value, isZero := sch.PrioritizedPrimaryField.ValueOf(ctx, v)
	if !isZero {
		e.id = toString(value)
	} else {
		// с пустыми идентификаторами не создаем
		return nil
	}

	return e
}

func (e *entity) GetKey() string {
	return getEntityKey(e.schema.Table, e.primaryKey, e.id)
}

func (e *entity) GetOtherKeys() (keys []string) {
	for _, pk := range e.otherPrimaryKeys {
		if pk == e.primaryKey {
			continue
		}
		v, ok := e.fields[pk]
		if ok {
			keys = append(keys, getEntityKey(e.schema.Table, pk, v))
		}
	}
	return
}

func (e *entity) Value() any {
	return e.reflectValue.Interface()
}

func (e *entity) Sync(ctx context.Context) *entity {
	if e == nil {
		return nil
	}

	e.fields = make(map[string]string)
	for _, f := range e.schema.Fields {
		if f.DBName == "" ||
			!f.Updatable ||
			f.AutoUpdateTime > 0 ||
			f.PrimaryKey {
			continue
		}
		value, _ := f.ValueOf(ctx, e.reflectValue)
		e.fields[f.DBName] = toString(value)
	}

	return e
}

func getEntityKey(table, column, value string) string {
	return fmt.Sprintf("%s.%s=%s", table, column, value)
}
