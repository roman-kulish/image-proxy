package roxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})
)

func TestMethodAllowed(t *testing.T) {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/dummy", nil)

	chain(handler, method(http.MethodGet)).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("expected HTTP Status Code %d, got %d", http.StatusOK, res.Code)
	}

	expected := http.StatusText(http.StatusOK)

	if !strings.Contains(res.Body.String(), expected) {
		t.Errorf("expected HTTP Body %s, got %s", expected, res.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/dummy", nil)

	chain(handler, method(http.MethodGet)).ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected HTTP Status Code %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	expected := http.StatusText(http.StatusMethodNotAllowed)

	if !strings.Contains(res.Body.String(), expected) {
		t.Errorf("expected HTTP Body %s, got %s", expected, res.Body.String())
	}
}

func TestLogger(t *testing.T) {
	res := httptest.NewRecorder()

	req, _ := http.NewRequest(http.MethodPatch, "/dummy", nil)
	req.RemoteAddr = "192.168.0.1"
	req.RequestURI = "/dummy"

	now := time.Now().Format(timeFormat)
	w := new(bytes.Buffer)

	chain(handler, logger(w)).ServeHTTP(res, req)

	asserts := map[string]string{
		":remote-addr":    req.RemoteAddr,
		":datetime":       "[" + now + "]",
		":method":         req.Method,
		":url":            req.RequestURI,
		":http-version":   req.Proto,
		":status":         strconv.Itoa(http.StatusOK),
		":content-length": strconv.Itoa(len(http.StatusText(http.StatusOK))),
		":response-time":  "0.000 ms",
	}

	log := w.String()

	for k, v := range asserts {
		if !strings.Contains(log, v) {
			t.Errorf("expected %s to be %s in %s", k, v, log)
		}
	}
}
