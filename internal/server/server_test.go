package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/TechBuilder-360/Rental-API/internal/config"
	"github.com/TechBuilder-360/Rental-API/internal/graph/resolver"
)

func TestGraphQLHealthQuery(t *testing.T) {
	srv := New(config.AppConfig{Env: "test", Port: 0}, &resolver.Resolver{}, nil, zerolog.Nop())

	body := strings.NewReader(`{"query":"{ health { status } }"}`)
	req := httptest.NewRequest(http.MethodPost, graphqlPath, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if want := `{"data":{"health":{"status":"ok"}}}`; rec.Body.String() != want {
		t.Fatalf("unexpected body: got %s, want %s", rec.Body.String(), want)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := New(config.AppConfig{Env: "test"}, &resolver.Resolver{}, nil, zerolog.Nop())

	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
