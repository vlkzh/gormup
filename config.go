package gormup

type Config struct {
	Store               Store
	WithoutQueryCache   bool
	WithoutReduceUpdate bool
}
