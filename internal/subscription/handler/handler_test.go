package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"subscription-billing-api/internal/subscription/model"
	"subscription-billing-api/internal/subscription/repository"
	"subscription-billing-api/internal/subscription/service"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo, err := repository.NewJSONRepository(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	handler := New(service.New(repo))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func futureDate(t *testing.T) string {
	t.Helper()
	return time.Now().AddDate(0, 0, 14).Format(model.DateLayout)
}

func TestHandlerCreateRejectsUnknownFields(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	body := fmt.Sprintf(
		`{"name":"Netflix","amount":39.9,"status":"active","next_renewal_date":%q,"unknown":"x"}`,
		futureDate(t),
	)
	resp, err := http.Post(server.URL+"/api/v1/subscriptions", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", resp.StatusCode)
	}
}

func TestHandlerListInvalidStatus(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/subscriptions?status=invalid")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", resp.StatusCode)
	}
}

func TestHandlerUpdateRenewalDateNotFound(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	body := fmt.Sprintf(`{"next_renewal_date":%q}`, futureDate(t))
	req, err := http.NewRequest(
		http.MethodPatch,
		server.URL+"/api/v1/subscriptions/missing/renewal-date",
		bytes.NewBufferString(body),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing subscription, got %d", resp.StatusCode)
	}
}

func TestHandlerCreateReturnsMetadata(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	body := fmt.Sprintf(
		`{"name":"Netflix","amount":39.9,"status":"","next_renewal_date":%q}`,
		futureDate(t),
	)
	resp, err := http.Post(server.URL+"/api/v1/subscriptions", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var envelope struct {
		Data model.Subscription `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Data.Status != model.StatusActive {
		t.Fatalf("expected active status, got %q", envelope.Data.Status)
	}
	if envelope.Data.Metadata == nil || envelope.Data.Metadata["source"] != "api" {
		t.Fatalf("expected source metadata in response, got %#v", envelope.Data.Metadata)
	}
}
