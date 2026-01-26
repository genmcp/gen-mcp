package health

import (
	"net/http"
	"sync/atomic"
)

type Checker interface {
	SetReady(ready bool)
	LivenessHandler(w http.ResponseWriter, r *http.Request)
	ReadinessHandler(w http.ResponseWriter, r *http.Request)
}

type checker struct {
	ready atomic.Bool
}

var _ Checker = &checker{}

func NewChecker() Checker {
	return &checker{}
}

func (c *checker) SetReady(ready bool) {
	c.ready.Store(ready)
}

func (c *checker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (c *checker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if c.ready.Load() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("not ready"))
}
