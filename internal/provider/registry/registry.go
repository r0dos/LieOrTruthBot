package registry

import (
	"context"
	"errors"
	"sync"
)

type execFunc func(v string)

type entry struct {
	cancelFunc context.CancelFunc
	checkFunc  execFunc
}

type Registry struct {
	baseCtx context.Context
	reg     map[string]*entry
	mu      sync.RWMutex
}

func NewRegistry(ctx context.Context) *Registry {
	return &Registry{
		baseCtx: ctx,
		reg:     make(map[string]*entry),
	}
}

func (r *Registry) Add(key, code string) (context.Context, error) {
	var answer string

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.reg[key]; ok {
		return nil, errors.New("entry already exists")
	}

	ctxWC, cancel := context.WithCancel(r.baseCtx)

	r.reg[key] = &entry{
		cancelFunc: cancel,
		checkFunc: func(v string) {
			answer += v
			if answer == code {
				cancel()

				return
			}

			if len(answer) == len(code) {
				answer = answer[len(answer)-1:]
			}
		},
	}

	return ctxWC, nil
}

func (r *Registry) Update(key, code string) error {
	var answer string

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.reg[key]; !ok {
		return errors.New("entry not found")
	}

	r.reg[key].checkFunc = func(v string) {
		answer += v
		if answer == code {
			r.reg[key].cancelFunc()

			return
		}

		if len(answer) == len(code) {
			answer = answer[len(answer)-1:]
		}
	}

	return nil
}

func (r *Registry) Removal(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.reg[key]; !ok {
		return errors.New("entry not found")
	}

	delete(r.reg, key)

	return nil
}

func (r *Registry) Send(key, v string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.reg[key]
	if !ok {
		return errors.New("function not found")
	}

	e.checkFunc(v)

	return nil
}
