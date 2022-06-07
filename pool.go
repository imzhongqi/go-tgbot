package tgbot

import (
	"github.com/panjf2000/ants/v2"
)

type Pool interface {
	IsClosed() bool
	Go(f func()) error
}

type workerPool struct {
	p *ants.Pool
}

func NewAntsPool(p *ants.Pool) Pool {
	return &workerPool{
		p: p,
	}
}

func (wp *workerPool) IsClosed() bool {
	return wp.p.IsClosed()
}

func (wp *workerPool) Go(f func()) error {
	return wp.p.Submit(f)
}
