package services

import "testing"

func TestAppRegistryValid(t *testing.T) {
	apps, err := AppRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) == 0 || len(apps) > 12 {
		t.Fatalf("registry should be a small list, got %d", len(apps))
	}
}

func TestParseAppRegistryRejects(t *testing.T) {
	ok := `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://example.com/x"}`
	if _, err := ParseAppRegistry([]byte("[" + ok + "]")); err != nil {
		t.Fatalf("valid entry rejected: %v", err)
	}
	tail := `{"id":"t","name":"T","monogram":"TT","group":"operate","environment":"tailnet","url":"https://{tailnet_host}:8443/"}`
	if _, err := ParseAppRegistry([]byte("[" + tail + "]")); err != nil {
		t.Fatalf("valid tailnet entry rejected: %v", err)
	}
	bad := map[string]string{
		"dup":         ok + "," + ok,
		"group":       `{"id":"a","name":"A","monogram":"AA","group":"x","environment":"public","url":"https://example.com"}`,
		"env":         `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"lan","url":"https://example.com"}`,
		"scheme":      `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"ftp://example.com"}`,
		"userinfo":    `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://u:p@example.com"}`,
		"token":       `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://example.com/?api_token=1"}`,
		"secret":      `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://example.com/?Secret=1"}`,
		"ip":          `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"http://192.0.2.5:8080"}`,
		"ts.net":      `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://box.example.ts.net"}`,
		"monogram":    `{"id":"a","name":"A","monogram":"A","group":"work","environment":"public","url":"https://example.com"}`,
		"tailnet-lit": `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"tailnet","url":"https://example.com"}`,
		"public-ph":   `{"id":"a","name":"A","monogram":"AA","group":"work","environment":"public","url":"https://{tailnet_host}/"}`,
	}
	for name, body := range bad {
		if _, err := ParseAppRegistry([]byte("[" + body + "]")); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := ParseAppRegistry([]byte("{")); err == nil {
		t.Error("bad json accepted")
	}
}
