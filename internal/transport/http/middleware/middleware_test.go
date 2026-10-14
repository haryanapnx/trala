package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDSetsHeaderAndContext(t *testing.T) {
	var fromCtx string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromCtx = RequestIDFrom(r.Context())
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEmpty(t, fromCtx)
	assert.Equal(t, fromCtx, rr.Header().Get("X-Request-ID"))
}

func TestRequestIDHonoursIncomingHeader(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc-123", RequestIDFrom(r.Context()))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc-123")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, "abc-123", rr.Header().Get("X-Request-ID"))
}

func TestLoggerRecordsStatusAndFields(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	h := RequestID(Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/teapot", nil))

	out := buf.String()
	assert.Contains(t, out, `"status":418`)
	assert.Contains(t, out, `"level":"WARN"`)
	assert.Contains(t, out, `"request_id"`)
	assert.Contains(t, out, `"latency_ms"`)
	assert.Contains(t, out, `"path":"/teapot"`)
}

func TestLoggerUsesErrorLevelFor5xx(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	h := Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Contains(t, buf.String(), `"level":"ERROR"`)
}

func TestRecoverWrites500Envelope(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	h := RequestID(Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body["code"])
	assert.Equal(t, float64(http.StatusInternalServerError), body["status"])
	assert.NotEmpty(t, body["timestamp"])

	// The panic detail is logged server-side but never sent to the client.
	assert.Contains(t, buf.String(), "boom")
	assert.NotContains(t, rr.Body.String(), "boom")
}
