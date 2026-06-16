package mocks

import "sync"

// CallRecord enregistre un appel de méthode.
type CallRecord struct {
	Method string
	Args   []any
}

// CallRecorder enregistre les appels de méthodes pour vérification dans les tests.
type CallRecorder struct {
	mu    sync.Mutex
	Calls []CallRecord
}

// Record enregistre un appel.
func (r *CallRecorder) Record(method string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls = append(r.Calls, CallRecord{Method: method, Args: args})
}

// Called retourne true si la méthode a été appelée au moins une fois.
func (r *CallRecorder) Called(method string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.Calls {
		if c.Method == method {
			return true
		}
	}
	return false
}

// CallCount retourne le nombre d'appels pour une méthode donnée.
func (r *CallRecorder) CallCount(method string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, c := range r.Calls {
		if c.Method == method {
			count++
		}
	}
	return count
}

// Reset efface tous les appels enregistrés.
func (r *CallRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls = nil
}
