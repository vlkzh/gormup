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

	valid  bool
	id     string
	fields map[string]string
}

func createEntity(
	ctx context.Context,
	sch *schema.Schema,
	v reflect.Value,
) *entity {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Type().Kind() != reflect.Struct {
		return nil
	}
	if len(sch.PrimaryFields) != 1 {
		return nil
	}

	e := &entity{
		schema:       sch,
		reflectValue: v,
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

func getEntityKey(table, id string) string {
	return fmt.Sprintf("%s-%s", table, id)
}

func (e *entity) Key() string {
	return getEntityKey(e.schema.Table, e.id)
}

func (e *entity) Value() any {
	return e.reflectValue.Interface()
}

func (e *entity) Snap(ctx context.Context) *entity {
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
