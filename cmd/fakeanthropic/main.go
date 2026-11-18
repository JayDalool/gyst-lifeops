//go:build browsercheck

// Command fakeanthropic is a local stand-in for the Anthropic Messages API,
// for browser checks only (#243). It is built only with -tags browsercheck,
// binds to loopback only and returns fictional foods.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

const foods = `{"items":[` +
	`{"name":"Test lentil stew","role":"main","est_grams":250,"confidence":0.8,"visible":true,"notes":"Fictional"},` +
	`{"name":"Test flatbread","role":"side","est_grams":null,"confidence":0.6,"visible":true,"notes":""}],` +
	`"uncertainties":["Fictional reply from the fake provider"],"questions":[]}`

func message(text string) string {
	b := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text)
	return `{"id":"msg_fake","type":"message","role":"assistant","model":"fake","stop_reason":"end_turn","content":[{"type":"text","text":"` + b + `"}],"usage":{"input_tokens":1,"output_tokens":1}}`
}

func main() {
	port := flag.Int("port", 0, "loopback port (0 picks one)")
	mode := flag.String("mode", "ok", "ok, credit or spend")
	flag.Parse()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		// Only the model name is printed, so a browser check can see which
		// model the app asked for. The image and prompt are never logged.
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &req)
		log.Printf("request: model=%q", req.Model)
		w.Header().Set("Content-Type", "application/json")
		switch *mode {
		case "credit":
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"billing_error","message":"fictional"}}`)
		case "spend":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified API usage limits (fictional)"}}`)
		default:
			if strings.Contains(string(body), "json_schema") {
				_, _ = io.WriteString(w, message(foods))
			} else {
				_, _ = io.WriteString(w, message("Fictional coach reply."))
			}
		}
	})
	fmt.Printf("http://%s/\n", ln.Addr())
	log.Fatal(http.Serve(ln, nil))
}
