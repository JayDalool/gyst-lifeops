package services

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"sync"
)

// App groups and environments (GYST-101). The registry is a committed,
// non-secret list of external tools. Tailnet entries write their host as the
// TailnetHostPlaceholder so no internal hostname is ever committed.
const (
	AppGroupWork     = "work"
	AppGroupOperate  = "operate"
	AppGroupLearning = "learning"

	AppEnvPublic  = "public"
	AppEnvTailnet = "tailnet"

	TailnetHostPlaceholder = "{tailnet_host}"
)

// AppGroups is the display order of the Apps page.
var AppGroups = []string{AppGroupWork, AppGroupOperate, AppGroupLearning}

type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Monogram    string `json:"monogram"`
	Group       string `json:"group"`
	Environment string `json:"environment"`
	URL         string `json:"url"`
}

//go:embed apps_registry.json
var appsRegistryJSON []byte

var (
	appsOnce sync.Once
	appsList []App
	appsErr  error
)

// AppRegistry parses and validates the embedded registry once.
func AppRegistry() ([]App, error) {
	appsOnce.Do(func() {
		appsList, appsErr = ParseAppRegistry(appsRegistryJSON)
	})
	return appsList, appsErr
}

// ParseAppRegistry parses and validates registry JSON.
func ParseAppRegistry(raw []byte) ([]App, error) {
	var apps []App
	if err := json.Unmarshal(raw, &apps); err != nil {
		return nil, fmt.Errorf("apps registry: %w", err)
	}
	seen := map[string]bool{}
	for _, a := range apps {
		if err := validateApp(a); err != nil {
			return nil, fmt.Errorf("apps registry %q: %w", a.ID, err)
		}
		if seen[a.ID] {
			return nil, fmt.Errorf("apps registry: duplicate id %q", a.ID)
		}
		seen[a.ID] = true
	}
	return apps, nil
}

func validateApp(a App) error {
	if a.ID == "" || a.Name == "" {
		return errors.New("id and name are required")
	}
	if n := len([]rune(a.Monogram)); n < 2 || n > 3 {
		return errors.New("monogram must be 2-3 characters")
	}
	switch a.Group {
	case AppGroupWork, AppGroupOperate, AppGroupLearning:
	default:
		return errors.New("unknown group")
	}
	switch a.Environment {
	case AppEnvPublic, AppEnvTailnet:
	default:
		return errors.New("unknown environment")
	}
	tailnet := a.Environment == AppEnvTailnet
	if tailnet != strings.Contains(a.URL, TailnetHostPlaceholder) {
		return errors.New("only tailnet apps use the host placeholder, and they must")
	}
	probe := strings.ReplaceAll(a.URL, TailnetHostPlaceholder, "tailnet-host.invalid")
	u, err := url.Parse(probe)
	if err != nil {
		return errors.New("bad url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url must be http(s)")
	}
	if u.User != nil {
		return errors.New("url must not carry userinfo")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.HasSuffix(host, ".ts.net") {
		return errors.New("url needs a plain host, no .ts.net literal")
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return errors.New("url must not use an IP address")
	}
	for key := range u.Query() {
		k := strings.ToLower(key)
		for _, bad := range []string{"token", "key", "secret", "password", "auth", "sig"} {
			if strings.Contains(k, bad) {
				return errors.New("url must not carry credentials in the query")
			}
		}
	}
	return nil
}
