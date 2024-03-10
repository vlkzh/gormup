package gormup

import (
	"context"
	"sync"
)

type Store interface {
	Set(ctx context.Context, key string, val any)
	Get(ctx context.Context, key string) (any, bool)
	Delete(ctx context.Context, key string)
}

type store struct {
	sync.Mutex

	resetRegistered bool

	values map[string]any
}

func NewStore() Store {
	return &store{}
}

func (s *store) Set(ctx context.Context, key string, val any) {
	s.resetOnDone(ctx)

	s.Lock()
	defer s.Unlock()

	if s.values == nil {
		s.values = map[string]any{}
	}

	s.values[key] = val
}

func (s *store) Get(ctx context.Context, key string) (any, bool) {
	s.Lock()
	defer s.Unlock()

	if s.values == nil {
		return nil, false
	}

	v, ok := s.values[key]
	return v, ok
}

func (s *store) Delete(ctx context.Context, key string) {
	s.Lock()
	defer s.Unlock()

	if s.values == nil {
		return
	}

	delete(s.values, key)
}

func (s *store) resetOnDone(ctx context.Context) {
	s.Lock()
	defer s.Unlock()

	if s.resetRegistered {
		return
	}

	s.resetRegistered = true
	go func() {
		<-ctx.Done()

		s.Lock()
		defer s.Unlock()

		s.values = nil
		s.resetRegistered = false
	}()
}
