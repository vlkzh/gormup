package gormup

import (
	"gorm.io/gorm"
)

func Register(db *gorm.DB, cfg Config) {
	if cfg.Store == nil {
		cfg.Store = NewStore()
	}
	pl := &plugin{
		config:   cfg,
		entities: newEntityStore(cfg.Store),
	}
	pl.register(db)
}

func WithoutQueryCache(db *gorm.DB) *gorm.DB {
	return db.Scopes(func(db *gorm.DB) *gorm.DB {
		return db.Set(withoutQueryCacheKey, true)
	})
}

func WithoutReduceUpdate(db *gorm.DB) *gorm.DB {
	return db.Scopes(func(db *gorm.DB) *gorm.DB {
		return db.Set(withoutReduceUpdateKey, true)
	})
}
