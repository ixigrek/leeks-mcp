package leekwars

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGetSendsBearerTokenAndReturnsBody(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/leek/get/1" {
			t.Errorf("chemin = %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	body, err := c.Get(context.Background(), "leek/get/1")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"id":1}` {
		t.Fatalf("corps = %s", body)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestGetOmitsAuthorizationWithoutToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if _, err := c.Get(context.Background(), "leek/get/1"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization inattendu : %q", gotAuth)
	}
}

func TestGetReportsAPIErrorWithoutLeakingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"error":"wrong_token"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-token")
	_, err := c.Get(context.Background(), "fight/get/1")
	if err == nil {
		t.Fatal("erreur attendue")
	}
	msg := err.Error()
	if !strings.Contains(msg, "fight/get/1") || !strings.Contains(msg, "wrong_token") {
		t.Fatalf("message incomplet : %s", msg)
	}
	if strings.Contains(msg, "secret-token") {
		t.Fatalf("le token fuite dans l'erreur : %s", msg)
	}
}

func TestGetReportsSuccessFalseEvenWithStatus200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"error":"fight_not_found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Get(context.Background(), "fight/get/1")
	if err == nil || !strings.Contains(err.Error(), "fight_not_found") {
		t.Fatalf("erreur = %v", err)
	}
}

func TestGetRateLimitsToFivePerSecond(t *testing.T) {
	var mu sync.Mutex
	var stamps []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		stamps = append(stamps, time.Now())
		mu.Unlock()
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	for i := 0; i < 10; i++ {
		if _, err := c.Get(context.Background(), "leek/get/1"); err != nil {
			t.Fatal(err)
		}
	}
	// 10 requêtes à 5/s : la dixième part au plus tôt 1,8 s après la première.
	if d := stamps[9].Sub(stamps[0]); d < 1700*time.Millisecond {
		t.Fatalf("10 requêtes en %v, limiteur inactif", d)
	}
}
