/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// A stalling endpoint used to block the caller forever: SendToHomerun built an
// http.Client with no Timeout and a request with no context.
func TestSendToHomerunContextRespectsCancellation(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // never answer until the test says so
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := SendToHomerunContext(ctx, srv.URL, "token", []byte(`{}`), false)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the context expires, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected a DeadlineExceeded error, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("call was not bounded by the context: took %v", elapsed)
	}
}

func TestSendToHomerunContextIsCancellableMidFlight(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := SendToHomerunContext(ctx, srv.URL, "token", []byte(`{}`), false)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected a Canceled error, got %v", err)
	}
}

// Every call used to build a fresh http.Transport, so no TCP or TLS connection
// was ever reused. On a service's hot path that is the dominant cost.
func TestSendToHomerunReusesConnections(t *testing.T) {
	SetHTTPClient(nil)
	t.Cleanup(func() { SetHTTPClient(nil) })

	var mu sync.Mutex
	conns := map[string]struct{}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		conns[r.RemoteAddr] = struct{}{}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const sends = 10
	for i := 0; i < sends; i++ {
		resp, err := SendToHomerun(srv.URL, "token", []byte(`{}`), false)
		if err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("send %d: unexpected status %d", i, resp.StatusCode)
		}
	}

	mu.Lock()
	got := len(conns)
	mu.Unlock()

	// One connection for all ten sends. Anything approaching `sends` means the
	// pool is not being reused.
	if got != 1 {
		t.Errorf("expected all %d sends over one connection, saw %d distinct client connections", sends, got)
	}
}

func TestDefaultClientCarriesATimeout(t *testing.T) {
	SetHTTPClient(nil)
	if got := httpClientFor(false).Timeout; got != DefaultHTTPTimeout {
		t.Errorf("expected the default client to be bounded by %v, got %v", DefaultHTTPTimeout, got)
	}
	if got := httpClientFor(true).Timeout; got != DefaultHTTPTimeout {
		t.Errorf("expected the insecure client to be bounded by %v, got %v", DefaultHTTPTimeout, got)
	}
	if httpClientFor(false) == httpClientFor(true) {
		t.Error("secure and insecure calls must not share a TLS configuration")
	}
	// Same configuration must hand back the same client, or connections are
	// still not pooled across calls.
	first, second := httpClientFor(false), httpClientFor(false)
	if first != second {
		t.Error("expected one client per configuration, got a new one per call")
	}
}

func TestSetHTTPClientOverridesAndRestores(t *testing.T) {
	t.Cleanup(func() { SetHTTPClient(nil) })

	custom := &http.Client{Timeout: 1234 * time.Millisecond}
	SetHTTPClient(custom)
	if httpClientFor(false) != custom || httpClientFor(true) != custom {
		t.Error("expected the injected client to be used for both configurations")
	}

	SetHTTPClient(nil)
	if httpClientFor(false) == custom {
		t.Error("expected SetHTTPClient(nil) to restore the built-in clients")
	}
}
