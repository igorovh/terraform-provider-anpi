package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrandingSendsNullForRemovedImages(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/branding" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write(b)
	}))
	defer server.Close()
	logo := "PHN2Zy8+"
	got, err := New(server.URL, "token", "test").PutBranding(context.Background(), &Branding{SiteName: "ACME", Logo: &logo})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, `"favicon":null`) || !strings.Contains(body, `"logo":"PHN2Zy8+"`) {
		t.Fatalf("request body %s", body)
	}
	if got.Favicon != nil || got.Logo == nil || *got.Logo != logo {
		out, _ := json.Marshal(got)
		t.Fatalf("decoded %s", out)
	}
}
