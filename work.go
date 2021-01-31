package workgroup

import (
	"context"
	"sync"
)

// Ctx is an alias for the standard `context.Context`
type Ctx = context.Context

// Worker is a function that performs work
type Worker func(Ctx) error

// WorkerIdx is a function that performs work with for a given index
type WorkerIdx func(Ctx, int) error

// Work arranges for a group of workers to be executed
// and waits for these workers to complete before returning.
// The executer, e, responsible for executing these workers
// with various levels of concurrancy. The manager, m, determines
// when the context will be cancelled and which error is returned.
// If executer, e, is not provided then the DefaultExecuter() function
// is called for obtain the default. If manager, m, is not provied
// then function DefaultManager() is called be obtain the default manager.
func Work(ctx Ctx, e Executer, m Manager, g ...Worker) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	wg.Add(len(g))
	for _, w := range g {
		worker := w
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = worker(ctx)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

// Group returns a worker that immediately calls the
// Work() function to execute the given group of workers.
func Group(e Executer, m Manager, g ...Worker) Worker {
	return func(ctx Ctx) error {
		return Work(ctx, e, m, g...)
	}
}

// WorkFor arranges for the worker, w, to be executed n times
// and waits for these workers to complete before returning.
// See documention for Work() for details.
func WorkFor(ctx Ctx, n int, e Executer, m Manager, w WorkerIdx) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	wg.Add(n)
	for i := 0; i < n; i++ {
		index := i
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = w(ctx, index)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

// GroupFor returns a worker that immediately calls the
// WorkFor() function to execute the worker n times.
func GroupFor(n int, e Executer, m Manager, w WorkerIdx) Worker {
	return func(ctx Ctx) error {
		return WorkFor(ctx, n, e, m, w)
	}
}

// WorkChan arranges for the group of workers provided by channel, g,
// to be executed and waits for the channel to be closed and all
// workers to complete. See documention for Work() for details.
func WorkChan(ctx Ctx, e Executer, m Manager, g <-chan Worker) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	for w := range g {
		wg.Add(1)
		worker := w
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = worker(ctx)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

// GroupChan returns a worker that immediately calls
// WorkChan to execute the group of workers provided
// by the channel.
func GroupChan(e Executer, m Manager, g <-chan Worker) Worker {
	return func(ctx Ctx) error {
		return WorkChan(ctx, e, m, g)
	}
}