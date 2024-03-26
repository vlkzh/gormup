package gormup

import (
	"context"
)

type entityStore struct {
	store Store
}

func newEntityStore(store Store) *entityStore {
	return &entityStore{
		store: store,
	}
}

func (s *entityStore) Set(ctx context.Context, ent *entity) {
	s.store.Set(ctx, ent.GetKey(), ent)
	for _, key := range ent.GetOtherKeys() {
		s.store.Set(ctx, key, ent)
	}
}

func (s *entityStore) Get(ctx context.Context, key string) *entity {
	v, ok := s.store.Get(ctx, key)
	if !ok {
		return nil
	}
	ent, _ := v.(*entity)
	return ent
}

func (s *entityStore) GetByFieldValue(ctx context.Context, table, filed, val string) *entity {
	return s.Get(ctx, getEntityKey(table, filed, val))
}

func (s *entityStore) Delete(ctx context.Context, key string) {
	ent := s.Get(ctx, key)
	if ent == nil {
		return
	}
	s.store.Delete(ctx, key)
	for _, k := range ent.GetOtherKeys() {
		s.store.Delete(ctx, k)
	}
}
