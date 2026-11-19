package config

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// readsNames runs Load with a getenv that records every name it asks for.
func readsNames(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	getenv := func(k string) string { seen[k] = true; return "" }
	if _, err := Load(nil, getenv, noFiles); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// composeApp returns the environment names the gyst-lifeops service gets, with
// the raw line, so a hard-coded value can be told from a pass-through.
func composeApp(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile("../../docker-compose.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "  gyst-lifeops:")
	end := strings.Index(text, "  caddy:")
	if start < 0 || end < start {
		t.Fatal("docker-compose.yml: gyst-lifeops service not found")
	}
	if strings.Contains(text[start:end], "env_file:") {
		t.Fatal("the service uses env_file:, which would forward every variable; list them under environment: instead")
	}
	out := map[string]string{}
	re := regexp.MustCompile(`(?m)^\s+- ([A-Z][A-Z0-9_]*)=(.*)$`)
	for _, m := range re.FindAllStringSubmatch(text[start:end], -1) {
		out[m[1]] = m[2]
	}
	return out
}

// Compose and .env.example must not drift from what config.Load reads. A
// variable Load reads is either forwarded as ${NAME:-}, fixed on purpose, or
// (the NAME_FILE forms) explicitly not forwarded and said so in .env.example.
func TestComposeAndExampleMatchTheVariablesLoadReads(t *testing.T) {
	fixed := map[string]string{"PORT": "8080", "DB_PATH": "/app/data/lifeops.db"}
	compose := composeApp(t)
	example, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	exampleText := string(example)

	read := map[string]bool{}
	for _, name := range readsNames(t) {
		read[name] = true
		if strings.HasSuffix(name, "_FILE") {
			if _, ok := compose[name]; ok {
				t.Errorf("%s is forwarded by Compose; the NAME_FILE forms need a mount, so decide that in its own change and update this test", name)
			}
			continue
		}
		if !strings.Contains("\n"+exampleText, "\n"+name+"=") {
			t.Errorf(".env.example does not list %s", name)
		}
		line, ok := compose[name]
		switch {
		case !ok:
			t.Errorf("docker-compose.yml does not pass %s to the container", name)
		case fixed[name] != "":
			if line != fixed[name] {
				t.Errorf("%s in Compose is %q, want the fixed %q", name, line, fixed[name])
			}
		case line != "${"+name+":-}":
			t.Errorf("%s in Compose is %q, want the plain pass-through ${%s:-}", name, line, name)
		}
	}
	for name := range compose {
		if !read[name] {
			t.Errorf("docker-compose.yml passes %s, which config.Load never reads", name)
		}
	}
	if !strings.Contains(exampleText, "NAME_FILE") || !strings.Contains(exampleText, "NOT forward") {
		t.Error(".env.example must say plainly that Compose does not forward NAME_FILE variables")
	}
}
