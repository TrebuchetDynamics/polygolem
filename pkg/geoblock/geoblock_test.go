package geoblock

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckDecodesVerdict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s; want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blocked":true,"ip":"1.2.3.4","country":"US","region":"NY"}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL, srv.Client()).Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	want := Result{Blocked: true, IP: "1.2.3.4", Country: "US", Region: "NY"}
	if got != want {
		t.Fatalf("Check = %+v; want %+v", got, want)
	}
}

func TestCheckRejectsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := New(srv.URL, srv.Client()).Check(context.Background()); err == nil {
		t.Fatal("expected error on 403")
	}
}

func TestNewDefaults(t *testing.T) {
	c := New("", nil)
	if c.baseURL != DefaultURL {
		t.Fatalf("baseURL = %q; want %q", c.baseURL, DefaultURL)
	}
	if c.httpClient == nil {
		t.Fatal("httpClient not defaulted")
	}
}
