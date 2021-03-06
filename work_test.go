package workgroup

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSimpleWork(t *testing.T) {

	counts := make([]int, 10000)
	workers := make([]Worker, len(counts))
	for i := 0; i < len(counts); i++ {
		index := i
		workers[i] = func(ctx context.Context) error {
			time.Sleep(time.Millisecond)
			counts[index]++
			return nil
		}
	}

	Work(nil, nil, nil, workers...)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}
}

func TestSimpleWorkFor(t *testing.T) {

	counts := make([]int, 10000)

	WorkFor(nil, nil, nil, len(counts),
		func(ctx context.Context, index int) error {
			time.Sleep(time.Millisecond)
			counts[index]++
			return nil
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}
}

func TestSimpleWorkChan(t *testing.T) {

	counts := make([]int, 10000)
	workers := make(chan Worker)
	go func() {
		for i := 0; i < len(counts); i++ {
			index := i
			workers <- func(ctx context.Context) error {
				time.Sleep(time.Millisecond)
				counts[index]++
				return nil
			}
		}
		close(workers)
	}()

	WorkChan(nil, nil, nil, workers)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}
}

func TestLimitedWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	WorkFor(context.Background(), NewLimited(8), CancelNeverFirstError(), len(counts),
		func(ctx context.Context, index int) error {
			select {
			case tokens <- struct{}{}:
				break
			default:
				t.Errorf("Worker %d must wait to send token", index)
			}

			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-tokens:
				break
			default:
				t.Errorf("Worker %d must wait to recieve token", index)
			}

			return nil
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Work %d has not completed", c)
		}
	}
}

func TestPoolWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	ctx, cancel := context.WithCancel(context.Background())

	WorkFor(ctx, NewPool(ctx, 8), CancelNeverFirstError(), len(counts),
		func(ctx context.Context, index int) error {
			select {
			case tokens <- struct{}{}:
				break
			default:
				t.Errorf("Worker %d must wait to send token", index)
			}

			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-tokens:
				break
			default:
				t.Errorf("Worker %d must wait to recieve token", index)
			}

			return nil
		},
	)

	cancel()

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Work %d has not completed", c)
		}
	}
}

func TestCancelNeverFirstError(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelNeverFirstError(),
	}

	err := WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index >= 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				t.Errorf("Work group context cancelled")
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err == nil {
		t.Errorf("Work group error is nil")
	}
	for _, e := range m.Errors {
		if e != nil {
			if e != err {
				t.Fatal("Work group error is not first error")
			}
			break
		}
	}
}

func TestCancelOnFirstError(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstError(),
	}

	err := WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index >= 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err == nil {
		t.Errorf("Work group error is nil")
	}
	errored := false
	for i, e := range m.Errors {
		if errored {
			if e == nil {
				t.Fatalf("Expecting accumulated error (%d) to not be nil", i)
			}
			if e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled", i)
			}
		} else {
			if e != nil {
				if e != err {
					t.Fatalf("Work group error is not first accumulated error (%d)", i)
				}
				errored = true
			}
		}
	}
}

func TestCancelOnFirstSuccess(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstSuccess(),
	}

	err := WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index < 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err != nil {
		t.Errorf("Work group error is not nil: %s", err)
	}
	success := false
	for i, e := range m.Errors {
		if success {
			if e == nil {
				t.Fatalf("Expecting accumulated error (%d) to not be nil", i)
			}
			if e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled", i)
			}
		} else {
			if e != nil {
				if e == context.Canceled {
					t.Fatalf("Expecting accumulated error (%d) to not be canceled", i)
				}
			} else {
				success = true
			}
		}
	}
}

func TestCancelOnFirstComplete(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstComplete(),
	}

	err := WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err != nil {
		t.Errorf("Work group error is not nil: %s", err)
	}
	for i, e := range m.Errors {
		if i > 0 {
			if e != nil && e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled: %s", i, e)
			}
		} else {
			if e != nil {
				t.Fatalf("Expecting accumulated error (%d) to be nil: %s", i, e)
			}
		}
	}
}

func TestRecoverManager(t *testing.T) {

	counts := make([]int, 10000)

	m := Recover(CancelOnFirstError())

	err := WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index == 500 {
				panic(fmt.Sprintf("worker %d failed", index))
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Fatalf("Worker %d has not completed", c)
		}
	}

	if err == nil {
		t.Fatal("Work group error is nil")
	}
	if err, ok := err.(*PanicError); ok {
		if err.Error() != "panic: worker 500 failed" {
			t.Fatalf("Work group panic error value incorrect")
		}
	} else {
		t.Errorf("Work group error is not a PanicError")
	}
}

func TestRepanicManager(t *testing.T) {

	counts := make([]int, 10000)

	defer func() {
		for _, c := range counts {
			if c != 1 {
				t.Fatalf("Worker %d has not completed", c)
			}
		}

		if v := recover(); v != nil {
			if v != "worker 500 failed" {
				t.Fatalf("Work group panic value incorrect")
			}
		} else {
			t.Fatalf("Work group did not panic")
		}
	}()

	m := Repanic(CancelOnFirstError())

	WorkFor(context.Background(), NewUnlimited(), m, len(counts),
		func(ctx context.Context, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index == 500 {
				panic(fmt.Sprintf("worker %d failed", index))
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)
}
