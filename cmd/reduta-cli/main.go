// Command reduta-cli is an operator tool. Today it verifies that a plugin
// correctly accepts signed webhooks (spec 7 / "reduta plugin verify").
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) >= 3 && args[0] == "plugin" && args[1] == "verify" {
		secret := "test-secret"
		url := args[2]
		for i := 3; i < len(args)-1; i++ {
			if args[i] == "--secret" {
				secret = args[i+1]
			}
		}
		os.Exit(verify(url, secret))
	}
	fmt.Println("usage: reduta-cli plugin verify <base_url> [--secret <secret>]")
	os.Exit(2)
}

func sign(secret, ts, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func post(url, event, delivery, ts, sig, body string) (int, error) {
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(url, "/")+"/hooks", bytes.NewReader([]byte(body)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Reduta-Event", event)
	req.Header.Set("X-Reduta-Delivery", delivery)
	req.Header.Set("X-Reduta-Timestamp", ts)
	req.Header.Set("X-Reduta-Signature", sig)
	cl := &http.Client{Timeout: 3 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func verify(url, secret string) int {
	body := `{"type":"tick.minute","ts":"2026-01-01T00:00:00Z"}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := sign(secret, ts, body)
	pass := 0
	total := 0
	check := func(name string, ok bool, detail string) {
		total++
		if ok {
			pass++
			fmt.Printf("  PASS %s\n", name)
		} else {
			fmt.Printf("  FAIL %s (%s)\n", name, detail)
		}
	}

	fmt.Printf("verifying plugin at %s\n", url)
	code, err := post(url, "tick.minute", "delivery-1", ts, sig, body)
	check("accepts signed webhook (2xx)", err == nil && code >= 200 && code < 300, fmt.Sprintf("err=%v code=%d", err, code))

	code, err = post(url, "tick.minute", "delivery-1", ts, sig, body)
	check("accepts duplicate delivery (idempotent 2xx)", err == nil && code >= 200 && code < 300, fmt.Sprintf("err=%v code=%d", err, code))

	code, err = post(url, "tick.minute", "delivery-2", ts, "sha256=deadbeef", body)
	check("rejects bad signature (4xx)", err == nil && code >= 400 && code < 500, fmt.Sprintf("err=%v code=%d", err, code))

	fmt.Printf("%d/%d checks passed\n", pass, total)
	if pass == total {
		return 0
	}
	return 1
}
