package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSORequests(t *testing.T) {
	var saved SSO
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sso" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(saved)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(saved)
		case http.MethodDelete:
			saved.Enabled = false
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	c := New(server.URL, "token", "test")
	want := &SSO{Enabled: true, BaseURL: "https://status.example", Issuer: "https://id.example", ClientID: "anpi", ClientSecret: "secret", RequiredRole: "admin", Scopes: "openid profile email"}
	got, err := c.PutSSO(context.Background(), want)
	if err != nil || *got != *want {
		t.Fatalf("PUT: got %#v, error %v", got, err)
	}
	got, err = c.GetSSO(context.Background())
	if err != nil || *got != *want {
		t.Fatalf("GET: got %#v, error %v", got, err)
	}
	if err := c.DeleteSSO(context.Background()); err != nil {
		t.Fatal(err)
	}
	if saved.Enabled {
		t.Fatal("DELETE did not disable SSO")
	}
}
