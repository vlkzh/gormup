package gormup

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
)

const (
	supportKey             = "gormup:support"
	entityKey              = "gormup:entity"
	withoutQueryCacheKey   = "gormup:without_query_cache"
	withoutReduceUpdateKey = "gormup:without_reduce_update"
)

var ErrNotChanged = errors.New("not changed")
var ErrAlreadyFetched = errors.New("already fetched")

type plugin struct {
	config   Config
	entities *entityStore
}

func (p *plugin) register(db *gorm.DB) {
	queryCallback := db.Callback().Query()
	queryCallback.Before("gorm:query").Register("gormup:before_query", p.beforeQuery)
	queryCallback.After("gorm:query").Register("gormup:after_query", p.afterQuery)
	queryCallback.After("*").Register("gormup:after_all_query", p.afterAllQuery)

	updateCallback := db.Callback().Update()
	updateCallback.Before("gorm:update").Register("gormup:before_update", p.beforeUpdate)
	updateCallback.After("gorm:update").Register("gormup:after_update", p.afterUpdate)

	deleteCallback := db.Callback().Update()
	deleteCallback.Before("*").Register("gormup:before_delete", p.beforeDelete)
}

func (p *plugin) withoutQueryCache(db *gorm.DB) bool {
	if p.config.WithoutQueryCache {
		return true
	}
	return p.getBool(db, withoutQueryCacheKey, false)
}

func (p *plugin) withoutReduceUpdate(db *gorm.DB) bool {
	if p.config.WithoutReduceUpdate {
		return true
	}
	return p.getBool(db, withoutReduceUpdateKey, false)
}

func (p *plugin) isSupportSelect(db *gorm.DB) bool {
	if db.Error != nil {
		return false
	}
	if db.Statement == nil ||
		db.Statement.Schema == nil ||
		len(db.Statement.Schema.PrimaryFields) > 1 {
		return false
	}

	if db.Statement.SkipHooks {
		return false
	}

	isSelectAll := len(db.Statement.Selects) == 0 ||
		slices.Contains(db.Statement.Selects, "*")
	if !isSelectAll {
		return false
	}

	modelType, _ := p.modelType(db.Statement.Dest)
	if modelType.String() != db.Statement.Schema.ModelType.String() {
		return false
	}

	return true
}

func (p *plugin) beforeQuery(db *gorm.DB) {
	if !p.isSupportSelect(db) {
		return
	}

	ids, ok := p.extractIds(db.Statement)
	if !ok {
		return
	}

	db.Set(supportKey, true)

	if p.withoutQueryCache(db) {
		return
	}

	ctx := db.Statement.Context

	values := make([]reflect.Value, len(ids))
	for i, id := range ids {
		ent := p.entities.Get(ctx, getEntityKey(db.Statement.Schema.Table, id))
		if ent != nil {
			values[i] = ent.reflectValue
		} else {
			return
		}
	}

	if len(values) == 0 {
		return
	}

	_, isArray := p.modelType(db.Statement.Dest)
	if isArray {
		newVal := reflect.MakeSlice(db.Statement.ReflectValue.Type(), len(values), len(values))
		for i, v := range values {
			refEl := newVal.Index(i)
			if refEl.Kind() == reflect.Ptr {
				n := reflect.New(refEl.Type().Elem())
				n.Elem().Set(v)
				refEl.Set(n)
			} else {
				refEl.Set(v)
			}
		}
		db.Statement.ReflectValue.Set(newVal)
	} else {
		db.Statement.ReflectValue.Set(values[0])
	}

	db.Error = ErrAlreadyFetched
}

func (p *plugin) afterQuery(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	_, sup := db.Get(supportKey)
	if !sup {
		return
	}

	ctx := db.Statement.Context

	ent := createEntity(
		ctx,
		db.Statement.Schema,
		db.Statement.ReflectValue,
	)
	if ent == nil {
		return
	}

	ent.Snap(ctx)

	p.entities.Set(ctx, ent)
}

func (p *plugin) afterAllQuery(db *gorm.DB) {
	_, sup := db.Get(supportKey)
	if !sup {
		return
	}

	db.Statement.Settings.Delete(supportKey)

	if errors.Is(db.Error, ErrAlreadyFetched) {
		db.Error = nil
		return
	}
}

func (p *plugin) beforeDelete(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}

	ids, ok := p.extractIds(db.Statement)
	if !ok {
		return
	}

	for _, id := range ids {
		p.entities.Delete(db.Statement.Context, getEntityKey(db.Statement.Schema.Table, id))
	}
}

func (p *plugin) modelType(dest any) (reflect.Type, bool) {
	val := reflect.ValueOf(dest)

	for {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		} else {
			break
		}
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		tp := val.Type().Elem()
		if tp.Kind() == reflect.Pointer {
			tp = tp.Elem()
		}
		return tp, true
	}

	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	return val.Type(), false
}

func (p *plugin) extractIds(st *gorm.Statement) ([]string, bool) {
	if len(st.Clauses) == 0 {
		return nil, false
	}

	var clauseWhere clause.Clause
	for _, cl := range st.Clauses {
		if cl.Name == "WHERE" {
			clauseWhere = cl
			break
		}
	}

	if clauseWhere.Name != "WHERE" {
		return nil, false
	}

	where, ok := clauseWhere.Expression.(clause.Where)
	if !ok {
		return nil, false
	}
	if len(where.Exprs) != 1 {
		return nil, false
	}

	primaryKey := st.Schema.PrioritizedPrimaryField.DBName
	table := st.Schema.Table

	switch expr := where.Exprs[0].(type) {
	case clause.Expr:
		if len(expr.Vars) != 1 {
			return nil, false
		}

		given := strings.ToLower(strings.ReplaceAll(expr.SQL, " ", ""))
		expect := strings.ToLower(fmt.Sprintf("%s=?", primaryKey))
		if given != expect {
			return nil, false
		}
		return []string{toString(expr.Vars[0])}, true
	case clause.IN:
		if len(expr.Values) != 1 {
			return nil, false
		}
		col, ok := expr.Column.(clause.Column)
		if !ok {
			return nil, false
		}
		if col.Table != clause.CurrentTable && col.Table != table {
			return nil, false
		}
		if col.Name != clause.PrimaryKey && col.Name != primaryKey {
			return nil, false
		}
		ids := make([]string, len(expr.Values))
		for i, v := range expr.Values {
			ids[i] = toString(v)
		}
		return ids, true
	case clause.Eq:
		col, ok := expr.Column.(clause.Column)
		if !ok {
			return nil, false
		}
		if col.Table != clause.CurrentTable && col.Table != table {
			return nil, false
		}
		if col.Name != clause.PrimaryKey && col.Name != primaryKey {
			return nil, false
		}
		return []string{toString(expr.Value)}, true
	}

	return nil, false
}

func (p *plugin) beforeUpdate(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	if db.Statement.SkipHooks {
		return
	}

	if p.withoutReduceUpdate(db) {
		return
	}

	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.UpdateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(180)
		db.Statement.AddClauseIfNotExists(clause.Update{})
		if _, ok := db.Statement.Clauses["SET"]; !ok {
			if set := callbacks.ConvertToAssignments(db.Statement); len(set) != 0 {
				defer delete(db.Statement.Clauses, "SET")
				set = p.reduceUpdateSet(db, set)
				if len(set) == 0 {
					_ = db.AddError(ErrNotChanged)
					return
				}
				db.Statement.AddClause(set)
			} else {
				return
			}
		}

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func (p *plugin) afterUpdate(db *gorm.DB) {
	if p.withoutReduceUpdate(db) {
		return
	}

	if errors.Is(db.Error, ErrNotChanged) {
		db.Error = nil
		db.RowsAffected = -1
	} else {
		p.getEntity(db).Snap(db.Statement.Context)
		p.deleteEntity(db)
	}
}

func (p *plugin) reduceUpdateSet(db *gorm.DB, set clause.Set) clause.Set {

	ctx := db.Statement.Context

	current := createEntity(ctx, db.Statement.Schema, db.Statement.ReflectValue)
	if current == nil {
		return set
	}

	original := p.entities.Get(ctx, current.Key())
	if original == nil {
		return set
	}
	original.reflectValue = current.reflectValue

	p.setEntity(db, original)

	sch := db.Statement.Schema

	var changedSet clause.Set
	var autoUpdateSet clause.Set
	for _, v := range set {
		if !isSupportForUpdate(v) {
			continue
		}
		f, ok := sch.FieldsByDBName[v.Column.Name]
		if !ok {
			continue
		}

		newVal := toString(v.Value)
		originalVal := original.fields[f.DBName]
		if newVal == originalVal {
			continue
		}

		if f.AutoUpdateTime > 0 {
			autoUpdateSet = append(autoUpdateSet, v)
		} else {
			changedSet = append(changedSet, v)
		}
	}

	if len(changedSet) > 0 && len(autoUpdateSet) > 0 {
		changedSet = append(changedSet, autoUpdateSet...)
	}

	return changedSet
}

func (p *plugin) getBool(db *gorm.DB, key string, def bool) bool {
	v, ok := db.Get(key)
	if !ok {
		return def
	}
	boolVal, ok := v.(bool)
	if !ok {
		return def
	}
	return boolVal
}

func (p *plugin) getEntity(db *gorm.DB) *entity {
	v, ok := db.Get(entityKey)
	if !ok {
		return nil
	}
	ent, _ := v.(*entity)
	return ent
}

func (p *plugin) setEntity(db *gorm.DB, ent *entity) {
	db.Set(entityKey, ent)
}

func (p *plugin) deleteEntity(db *gorm.DB) {
	db.Statement.Settings.Delete(entityKey)
}
