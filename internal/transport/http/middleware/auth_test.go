package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubVerifier struct {
	userID string
	err    error
}

func (s stubVerifier) Parse(string) (string, error) { return s.userID, s.err }

func TestAuthRejectsMissingToken(t *testing.T) {
	h := Auth(stubVerifier{userID: "u1"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	h := Auth(stubVerifier{err: errors.New("bad token")})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer nope")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthSetsUserIDInContext(t *testing.T) {
	var got string
	var ok bool

	h := Auth(stubVerifier{userID: "user-123"})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, ok = UserIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")

	h.ServeHTTP(httptest.NewRecorder(), req)

	assert.True(t, ok)
	assert.Equal(t, "user-123", got)
}
