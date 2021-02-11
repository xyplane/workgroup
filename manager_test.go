package workgroup

import "sync"

// AccumulateManager is used only for testing (for now)
type AccumulateManager struct {
	mutex   sync.Mutex
	manager Manager
	Errors  []error
}

func (m *AccumulateManager) Result() error {
	return m.manager.Result()
}

func (m *AccumulateManager) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	n := m.manager.Manage(ctx, c, idx, err)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for len(m.Errors) < n {
		m.Errors = append(m.Errors, nil)
	}
	m.Errors[n-1] = *err

	return n
}