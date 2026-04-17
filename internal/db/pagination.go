package db

import "github.com/uptrace/bun"

type DefaultPagination interface {
	DefaultLimit() int
	MaxLimit() int
}

type Pagination[T DefaultPagination] struct {
	def T

	limit  int
	offset int
}

func NewPagination[T DefaultPagination](limit, offset int) *Pagination[T] {
	//nolint:exhaustruct // zero value
	return &Pagination[T]{
		limit:  limit,
		offset: offset,
	}
}

func (p *Pagination[T]) Limit() int {
	if p == nil {
		var def T
		return def.DefaultLimit()
	}

	if p.limit <= 0 {
		p.limit = p.def.DefaultLimit()
	}
	if p.limit > p.def.MaxLimit() {
		p.limit = p.def.MaxLimit()
	}

	return p.limit
}

func (p *Pagination[T]) Offset() int {
	if p == nil {
		return 0
	}

	if p.offset < 0 {
		p.offset = 0
	}
	return p.offset
}

func (p *Pagination[T]) Apply(query *bun.SelectQuery) *bun.SelectQuery {
	return query.Limit(p.Limit()).Offset(p.Offset())
}
