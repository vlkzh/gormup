package gormup

import "context"

type entityStore struct {
	store Store
}

func newEntityStore(store Store) *entityStore {
	return &entityStore{
		store: store,
	}
}

func (s *entityStore) Set(ctx context.Context, ent *entity) {
	s.store.Set(ctx, ent.Key(), ent)
}

func (s *entityStore) Get(ctx context.Context, key string) *entity {
	v, ok := s.store.Get(ctx, key)
	if !ok {
		return nil
	}
	ent, _ := v.(*entity)
	return ent
}

func (s *entityStore) Delete(ctx context.Context, key string) {
	s.store.Delete(ctx, key)
}
