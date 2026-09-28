package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// validPrayerForm is exactly what the Settings page submits: the fields of the
// <form method="post" action="/settings/prayer"> in settings.html. The Timezone
// control is read-only and unnamed, so a browser never sends "timezone"; one is
// added here to prove a crafted value cannot override the application timezone.
func validPrayerForm() url.Values {
	return url.Values{
		"latitude":       {"43.6532"},
		"longitude":      {"-79.3832"},
		"timezone":       {"Asia/Tokyo"},
		"method":         {"mwl"},
		"asr_method":     {"hanafi"},
		"alert_sound":    {"chime"},
		"athan_file":     {"athan.mp3"},
		"adjust_fajr":    {"2"},
		"adjust_dhuhr":   {"0"},
		"adjust_asr":     {"-3"},
		"adjust_maghrib": {"1"},
		"adjust_isha":    {"5"},
	}
}

// postPrayerSettings sends the form through the real middleware (session and
// same-origin guard) and route, as a signed-in browser does.
func postPrayerSettings(t *testing.T, h *Handler, cookie *http.Cookie, values url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/settings/prayer", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Origin", "http://"+request.Host)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)
	return recorder
}

func getSettings(t *testing.T, h *Handler, cookie *http.Cookie) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/settings", nil)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /settings = %d: %s", recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

func TestPrayerSettingsSaveFromSettingsPage(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)
	if page := getSettings(t, h, cookie); !strings.Contains(page, `action="/settings/prayer"`) {
		t.Fatal("GET /settings does not render the prayer form")
	}

	recorder := postPrayerSettings(t, h, cookie, validPrayerForm(), nil)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/settings#prayer" {
		t.Fatalf("POST /settings/prayer = %d %q: %s", recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
	}

	config, err := h.Service.PrayerConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Latitude != 43.6532 || config.Longitude != -79.3832 || config.Method != "mwl" || config.AsrMethod != "hanafi" ||
		config.AlertSound != "chime" || config.AthanFile != "athan.mp3" ||
		config.Adjustments["fajr"] != 2 || config.Adjustments["asr"] != -3 || config.Adjustments["maghrib"] != 1 || config.Adjustments["isha"] != 5 {
		t.Fatalf("prayer settings not persisted: %#v", config)
	}
	// The application timezone stays authoritative; the posted value is
	// neither used nor stored.
	if config.Timezone != "America/Winnipeg" {
		t.Fatalf("prayer timezone = %q; want the application timezone", config.Timezone)
	}
	if stored, _ := h.Service.Repo.GetSetting("prayer.timezone"); stored != "" {
		t.Fatalf("posted prayer timezone was stored: %q", stored)
	}

	page := getSettings(t, h, cookie)
	for _, want := range []string{`value="43.6532"`, `value="-79.3832"`, `<option value="mwl" selected>`, `<option value="hanafi" selected>`, `value="athan.mp3"`, `<input value="America/Winnipeg" readonly>`} {
		if !strings.Contains(page, want) {
			t.Errorf("GET /settings after save is missing %s", want)
		}
	}
	if strings.Contains(page, `name="timezone"`) {
		t.Error("Settings must not submit an editable timezone")
	}
}

func TestPrayerSettingsKeepConfiguredApplicationTimezone(t *testing.T) {
	h := authHandler(t)
	toronto, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Fatal(err)
	}
	h.Service.Loc = toronto // as if GYST_TIMEZONE=America/Toronto
	cookie := signIn(t, h)

	if recorder := postPrayerSettings(t, h, cookie, validPrayerForm(), nil); recorder.Code != http.StatusSeeOther {
		t.Fatalf("POST /settings/prayer = %d: %s", recorder.Code, recorder.Body.String())
	}
	if config, _ := h.Service.PrayerConfig(); config.Timezone != "America/Toronto" {
		t.Fatalf("prayer timezone = %q; want GYST_TIMEZONE", config.Timezone)
	}
	if !strings.Contains(getSettings(t, h, cookie), `<input value="America/Toronto" readonly>`) {
		t.Fatal("Settings does not show the configured application timezone")
	}
}

// Invalid input is rejected with 400 and nothing is written: the settings
// saved before stay exactly as they were.
func TestPrayerSettingsRejectInvalidInputWithoutWriting(t *testing.T) {
	cases := map[string][2]string{
		"latitude too large":   {"latitude", "91"},
		"latitude not number":  {"latitude", "north"},
		"longitude too small":  {"longitude", "-181"},
		"longitude empty":      {"longitude", ""},
		"adjustment too large": {"adjust_isha", "121"},
		"adjustment not int":   {"adjust_fajr", "1.5"},
		"unknown method":       {"method", "custom"},
		"athan path":           {"athan_file", "../secret.mp3"},
	}
	for name, field := range cases {
		t.Run(name, func(t *testing.T) {
			h := authHandler(t)
			cookie := signIn(t, h)
			if recorder := postPrayerSettings(t, h, cookie, validPrayerForm(), nil); recorder.Code != http.StatusSeeOther {
				t.Fatalf("initial save = %d: %s", recorder.Code, recorder.Body.String())
			}
			before, _ := h.Service.PrayerConfig()

			values := validPrayerForm()
			values.Set("latitude", "10")
			values.Set("adjust_dhuhr", "7")
			values.Set(field[0], field[1])
			if recorder := postPrayerSettings(t, h, cookie, values, nil); recorder.Code != http.StatusBadRequest {
				t.Fatalf("invalid %s=%q: status %d; want 400", field[0], field[1], recorder.Code)
			}
			after, _ := h.Service.PrayerConfig()
			if after.Latitude != before.Latitude || after.Longitude != before.Longitude || after.Method != before.Method ||
				after.AthanFile != before.AthanFile || after.Adjustments["dhuhr"] != before.Adjustments["dhuhr"] ||
				after.Adjustments["isha"] != before.Adjustments["isha"] || after.Adjustments["fajr"] != before.Adjustments["fajr"] {
				t.Fatalf("an invalid form changed saved settings: before %#v after %#v", before, after)
			}
		})
	}
}

// The same-origin guard still rejects a cross-site post, and nothing is saved.
func TestPrayerSettingsCrossSitePostIsRejected(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"sec-fetch-site": {"Sec-Fetch-Site": "cross-site"},
		"foreign origin": {"Origin": "https://untrusted.example"},
	} {
		t.Run(name, func(t *testing.T) {
			h := authHandler(t)
			cookie := signIn(t, h)
			if recorder := postPrayerSettings(t, h, cookie, validPrayerForm(), headers); recorder.Code != http.StatusForbidden {
				t.Fatalf("cross-site POST = %d; want 403", recorder.Code)
			}
			if stored, _ := h.Service.Repo.GetSetting("prayer.latitude"); stored != "" {
				t.Fatalf("cross-site POST saved latitude %q", stored)
			}
		})
	}
}
