package health

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	assert.NotNil(t, c)
}

func TestLivenessHandler_AlwaysReturns200(t *testing.T) {
	c := NewChecker()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	c.LivenessHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestReadinessHandler_Returns503WhenNotReady(t *testing.T) {
	c := NewChecker()
	// Default state is not ready

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	c.ReadinessHandler(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "not ready", rec.Body.String())
}

func TestReadinessHandler_Returns200WhenReady(t *testing.T) {
	c := NewChecker()
	c.SetReady(true)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	c.ReadinessHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestSetReady_TogglesState(t *testing.T) {
	c := NewChecker()

	// Initially not ready
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	c.ReadinessHandler(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	// Set ready
	c.SetReady(true)
	rec = httptest.NewRecorder()
	c.ReadinessHandler(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Set not ready again
	c.SetReady(false)
	rec = httptest.NewRecorder()
	c.ReadinessHandler(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReadinessHandler_ThreadSafety(t *testing.T) {
	c := NewChecker()

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
				rec := httptest.NewRecorder()
				c.ReadinessHandler(rec, req)
				// Just ensure no panic - status can be either
				assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusServiceUnavailable)
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.SetReady(j%2 == 0)
			}
		}()
	}

	wg.Wait()
}
