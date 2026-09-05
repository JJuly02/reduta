// Command koth-plugin is a reference King of the Hill plugin for Reduta. It
// speaks the Plugin API v1 (spec 7): it verifies signed webhooks and, on each
// tick, awards points to the current holder of each target via the core awards
// endpoint. The client's existing KotH project is wrapped by pointing its
// holder detection at this service; see docs/adr/0010-koth-plugin.md.
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	secret    = env("KOTH_WEBHOOK_SECRET", "test-secret")
	coreURL   = env("KOTH_CORE_URL", "")       // e.g. http://server:8080
	pluginTok = env("KOTH_PLUGIN_TOKEN", "")   // plg_...
	eventID   = env("KOTH_EVENT_ID", "")
	holder    = env("KOTH_HOLDER_TEAM", "")    // demo: current crown holder team id
	target    = env("KOTH_TARGET", "target-1")

	seen   = map[string]bool{}
	seenMu sync.Mutex
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest", manifest)
	mux.HandleFunc("/hooks", hooks)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	srv := &http.Server{Addr: ":8090", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	println("koth-plugin listening on :8090")
	_ = srv.ListenAndServe()
}

func manifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
  "id":"koth","name":"King of the Hill","version":"1.0.0",
  "capabilities":["score_awards","challenge_type","ui_slots"],
  "events":["tick.minute","solve.created"],
  "ui":{"slots":["scoreboard.side"]}
}`))
}

func hooks(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	ts := r.Header.Get("X-Reduta-Timestamp")
	got := r.Header.Get("X-Reduta-Signature")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(body)))
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(got), []byte(want)) {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}
	// Deduplicate by delivery id (spec 7.2).
	if d := r.Header.Get("X-Reduta-Delivery"); d != "" {
		seenMu.Lock()
		dup := seen[d]
		seen[d] = true
		seenMu.Unlock()
		if dup {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"duplicate"}`))
			return
		}
	}
	if r.Header.Get("X-Reduta-Event") == "tick.minute" && coreURL != "" && pluginTok != "" && eventID != "" && holder != "" {
		go awardTick(ts)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// awardTick gives the current holder points for this tick, idempotent by ref_id.
func awardTick(ts string) {
	payload, _ := json.Marshal(map[string]any{
		"event_id": eventID, "team_id": holder, "points": 5,
		"ref_id": "tick:" + ts + ":" + target, "reason": "KotH tick " + target,
		"meta": map[string]string{"target": target},
	})
	req, err := http.NewRequest(http.MethodPost, coreURL+"/api/v1/plugin/v1/awards", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+pluginTok)
	req.Header.Set("Content-Type", "application/json")
	cl := &http.Client{Timeout: 3 * time.Second}
	if resp, err := cl.Do(req); err == nil {
		resp.Body.Close()
	}
}
