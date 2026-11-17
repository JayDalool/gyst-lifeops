package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"lifeops-tracker/internal/healthcoach"
)

// GYST-169 (#218). Every key in these tests is an obvious fake.
const fakeAIKey = "sk-test-FAKE0000000000000000000000000000"

type aiFixture struct {
	h       *Handler
	cookie  *http.Cookie
	logs    *bytes.Buffer
	keyPath string
}

func newAIFixture(t *testing.T) *aiFixture {
	t.Helper()
	h := authHandler(t)
	f := &aiFixture{h: h, logs: &bytes.Buffer{}, keyPath: filepath.Join(t.TempDir(), "ai_settings.key")}
	h.Logger = slog.New(slog.NewTextHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h.Service.AIKeyPath = f.keyPath
	f.cookie = signIn(t, h)
	return f
}

func (f *aiFixture) post(t *testing.T, fields url.Values, mutate ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/settings/ai", strings.NewReader(fields.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(f.cookie)
	for _, m := range mutate {
		m(req)
	}
	rec := httptest.NewRecorder()
	f.h.Authenticated().ServeHTTP(rec, req)
	return rec
}

func aiForm_(action, password, key string) url.Values {
	return url.Values{
		"action": {action}, "password": {password}, "api_key": {key},
		"provider": {"openai"}, "chat_model": {"gpt-test-1"}, "photo_model": {"gpt-test-vision"},
	}
}

// assertNoKey fails when the fake key (or the password) shows up in text.
func assertNoKey(t *testing.T, where, text string) {
	t.Helper()
	for _, secret := range []string{fakeAIKey, "FAKE0000"} {
		if strings.Contains(text, secret) {
			t.Errorf("%s contains the API key", where)
		}
	}
}

func (f *aiFixture) auditText(t *testing.T) (string, []string) {
	t.Helper()
	rows, err := f.h.Service.Repo.RecentAuditEvents(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	var all strings.Builder
	var actions []string
	for _, r := range rows {
		all.WriteString(r.At + r.ActorRef + r.Action + r.TargetRef + r.Outcome + r.Detail + r.RequestID)
		actions = append(actions, r.Action+"/"+r.Outcome+"/"+r.Detail)
	}
	return all.String(), actions
}

func TestAISettingsSaveEnablesCoachAndNeverShowsKey(t *testing.T) {
	f := newAIFixture(t)
	if f.h.coach() != nil {
		t.Fatal("Coach is on before any key is saved")
	}
	rec := f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings?ai=saved#ai" {
		t.Fatalf("save = %d %q: %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	assertNoKey(t, "save response", rec.Body.String()+rec.Header().Get("Location"))
	if f.h.coach() == nil {
		t.Fatal("Coach is still off after a key was saved")
	}
	if f.h.coach().Name() != "openai" {
		t.Errorf("provider = %q", f.h.coach().Name())
	}

	page := getSettings(t, f.h, f.cookie)
	assertNoKey(t, "settings page", page)
	for _, want := range []string{"Key saved on ", `value="gpt-test-1"`, `value="gpt-test-vision"`, `type="password"`, "Remove key"} {
		if !strings.Contains(page, want) {
			t.Errorf("settings page is missing %q", want)
		}
	}
	// A GET of the page never carries a key attribute for the key field.
	if strings.Contains(page, `name="api_key" type="password" maxlength="300" autocomplete="new-password" autocapitalize="off" spellcheck="false" value=`) {
		t.Error("the key input has a value")
	}
	if strings.Contains(page, testPassword) {
		t.Error("settings page contains the password")
	}

	assertNoKey(t, "logs", f.logs.String())
	audit, actions := f.auditText(t)
	assertNoKey(t, "audit rows", audit)
	if !strings.Contains(strings.Join(actions, " "), "settings.ai_key.set/success/") {
		t.Errorf("audit actions = %v, want settings.ai_key.set with no detail", actions)
	}

	// The key file is 0600 and 32 bytes; the database holds ciphertext only.
	info, err := os.Stat(f.keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode = %v, want 0600", info.Mode().Perm())
	}
	if info.Size() != 32 {
		t.Errorf("key file size = %d, want 32", info.Size())
	}
	row, err := f.h.Service.Repo.GetAISettings()
	if err != nil || row == nil {
		t.Fatalf("row: %v %v", row, err)
	}
	if bytes.Contains(row.KeyCiphertext, []byte("FAKE0000")) || len(row.KeyNonce) != 12 {
		t.Error("stored key is not ciphertext with a 12-byte nonce")
	}
}

func TestAISettingsWrongPasswordStoresNothing(t *testing.T) {
	f := newAIFixture(t)
	rec := f.post(t, aiForm_("save", "not the password", fakeAIKey))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password = %d, want 401", rec.Code)
	}
	assertNoKey(t, "refusal page", rec.Body.String())
	if !strings.Contains(rec.Body.String(), aiMsgPassword) {
		t.Error("refusal text missing")
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("a row was stored after a wrong password")
	}
	if _, err := os.Stat(f.keyPath); err == nil {
		t.Error("a key file was created after a wrong password")
	}
	if f.h.coach() != nil {
		t.Error("Coach turned on after a wrong password")
	}
	_, actions := f.auditText(t)
	if !strings.Contains(strings.Join(actions, " "), "reauth.failed/failed/") {
		t.Errorf("audit actions = %v, want reauth.failed", actions)
	}
	assertNoKey(t, "logs", f.logs.String())
}

func TestAISettingsRemoveNeedsPassword(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	rec := f.post(t, aiForm_("remove", "", ""))
	if rec.Code != http.StatusUnauthorized || f.h.coach() == nil {
		t.Errorf("remove without the password = %d, coach on = %v", rec.Code, f.h.coach() != nil)
	}
}

func TestAISettingsCrossOriginIs403(t *testing.T) {
	f := newAIFixture(t)
	for name, mutate := range map[string]func(*http.Request){
		"sec-fetch-site": func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		"origin":         func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
	} {
		rec := f.post(t, aiForm_("save", testPassword, fakeAIKey), mutate)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", name, rec.Code)
		}
	}
	// The handler enforces it too, not only the session middleware.
	req := httptest.NewRequest(http.MethodPost, "/settings/ai", strings.NewReader(aiForm_("save", testPassword, fakeAIKey).Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("direct handler status = %d, want 403", rec.Code)
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("a cross-origin post stored settings")
	}
}

func TestAISettingsRequiresSession(t *testing.T) {
	f := newAIFixture(t)
	f.cookie = &http.Cookie{Name: sessionCookieName, Value: "nope"}
	rec := f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if rec.Code == http.StatusSeeOther && rec.Header().Get("Location") == "/settings?ai=saved#ai" {
		t.Fatal("saved without a session")
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("settings stored without a session")
	}
}

func TestAISettingsCiphertextDiffersPerSave(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	first, _ := f.h.Service.Repo.GetAISettings()
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	second, _ := f.h.Service.Repo.GetAISettings()
	if bytes.Equal(first.KeyNonce, second.KeyNonce) || bytes.Equal(first.KeyCiphertext, second.KeyCiphertext) {
		t.Error("two saves of the same key produced the same nonce or ciphertext")
	}
	// Saving only the models keeps the key and its ciphertext.
	f.post(t, aiForm_("save", testPassword, ""))
	third, _ := f.h.Service.Repo.GetAISettings()
	if !bytes.Equal(second.KeyCiphertext, third.KeyCiphertext) || f.h.coach() == nil {
		t.Error("a models-only save changed or lost the key")
	}
}

func TestAISettingsTamperedCiphertextIsNotConfigured(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	row, _ := f.h.Service.Repo.GetAISettings()
	row.KeyCiphertext[0] ^= 0xff
	if err := f.h.Service.Repo.SaveAISettings(*row); err != nil {
		t.Fatal(err)
	}
	if f.h.coach() != nil {
		t.Error("Coach is on with a tampered ciphertext")
	}
	page := getSettings(t, f.h, f.cookie)
	if !strings.Contains(page, "A saved key can't be read. Enter it again.") {
		t.Error("settings page doesn't say the saved key can't be read")
	}
	assertNoKey(t, "settings page", page)
	if strings.Contains(page, "Key saved on") {
		t.Error("a tampered key is shown as saved")
	}
	// Entering the key again repairs it.
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if f.h.coach() == nil {
		t.Error("re-entering the key didn't recover")
	}
}

func TestAISettingsMissingKeyFileIsNotConfigured(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if err := os.Remove(f.keyPath); err != nil {
		t.Fatal(err)
	}
	if f.h.coach() != nil {
		t.Error("Coach is on without the data key file (a restored backup)")
	}
	if !strings.Contains(getSettings(t, f.h, f.cookie), "A saved key can't be read") {
		t.Error("settings page doesn't ask for the key again")
	}
}

func TestAISettingsEnvironmentWins(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if f.h.coach() == nil {
		t.Fatal("setup: Coach should be on from settings")
	}
	// The server environment names a provider: saved settings are ignored,
	// even though the environment's own key is missing (Coach stays off).
	f.h.AIEnvManaged = true
	if f.h.coach() != nil {
		t.Error("saved settings were used although the environment manages the provider")
	}
	// An environment provider is used as is.
	env := &envCoach{}
	f.h.Coach = env
	if f.h.coach() != env {
		t.Error("the environment provider was not used")
	}
	f.h.Coach = nil
	page := getSettings(t, f.h, f.cookie)
	if !strings.Contains(page, "Configured by the server environment") {
		t.Error("settings page doesn't say the environment configures the provider")
	}
	if strings.Count(page, "disabled") < 5 {
		t.Error("the AI form isn't disabled")
	}
	rec := f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if rec.Code != http.StatusConflict {
		t.Errorf("save under env management = %d, want 409", rec.Code)
	}
	rec = f.post(t, aiForm_("remove", testPassword, ""))
	if rec.Code != http.StatusConflict {
		t.Errorf("remove under env management = %d, want 409", rec.Code)
	}
}

func TestAISettingsRemoveDisablesCoachAtOnce(t *testing.T) {
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if f.h.coach() == nil {
		t.Fatal("setup")
	}
	rec := f.post(t, aiForm_("remove", testPassword, ""))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings?ai=removed#ai" {
		t.Fatalf("remove = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if f.h.coach() != nil {
		t.Error("Coach is still on after the key was removed")
	}
	row, _ := f.h.Service.Repo.GetAISettings()
	if row == nil || row.KeyCiphertext != nil || row.KeyNonce != nil || row.KeySavedAt != "" {
		t.Errorf("key columns not cleared: %+v", row)
	}
	if row.Provider != "openai" || row.ChatModel != "gpt-test-1" {
		t.Error("provider and models should be kept on remove")
	}
	page := getSettings(t, f.h, f.cookie)
	if !strings.Contains(page, "Not set.") || strings.Contains(page, "Remove key") {
		t.Error("page doesn't show the removed state")
	}
	audit, actions := f.auditText(t)
	assertNoKey(t, "audit", audit)
	if !strings.Contains(strings.Join(actions, " "), "settings.ai_key.remove/success/") {
		t.Errorf("audit actions = %v", actions)
	}
	assertNoKey(t, "logs", f.logs.String())
}

func TestAISettingsKeyFileFailureFailsCleanly(t *testing.T) {
	f := newAIFixture(t)
	f.h.Service.AIKeyPath = filepath.Join(t.TempDir(), "missing-dir", "ai_settings.key")
	rec := f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	assertNoKey(t, "error page", body)
	if !strings.Contains(body, "encryption key file") || strings.Contains(body, "missing-dir") {
		t.Error("error text should be fixed and never name the path")
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("a row was stored although the key file could not be created")
	}
	if f.h.coach() != nil {
		t.Error("Coach on after a failed save")
	}
	// No key path configured at all.
	f.h.Service.AIKeyPath = ""
	if rec := f.post(t, aiForm_("save", testPassword, fakeAIKey)); rec.Code != http.StatusBadRequest {
		t.Errorf("no key path: status = %d, want 400", rec.Code)
	}
}

func TestAISettingsLooseKeyFileIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no unix modes")
	}
	f := newAIFixture(t)
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	if err := os.Chmod(f.keyPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if f.h.coach() != nil {
		t.Error("a world-readable key file was trusted")
	}
	if rec := f.post(t, aiForm_("save", testPassword, fakeAIKey)); rec.Code != http.StatusBadRequest {
		t.Errorf("save with a loose key file = %d, want 400", rec.Code)
	}
}

func TestAISettingsValidation(t *testing.T) {
	f := newAIFixture(t)
	cases := map[string]url.Values{
		"bad provider": func() url.Values { v := aiForm_("save", testPassword, fakeAIKey); v.Set("provider", "evil"); return v }(),
		"model with space": func() url.Values {
			v := aiForm_("save", testPassword, fakeAIKey)
			v.Set("chat_model", "gpt 5")
			return v
		}(),
		"model too long": func() url.Values {
			v := aiForm_("save", testPassword, fakeAIKey)
			v.Set("photo_model", strings.Repeat("a", 65))
			return v
		}(),
		"model with html": func() url.Values { v := aiForm_("save", testPassword, fakeAIKey); v.Set("chat_model", "<b>"); return v }(),
		"key too short":   aiForm_("save", testPassword, "sk-short"),
		"key bad charset": aiForm_("save", testPassword, "sk-test-FAKE0000 00000000000000000000"),
		"key too long":    aiForm_("save", testPassword, "sk-"+strings.Repeat("a", 400)),
		"unknown action":  aiForm_("steal", testPassword, ""),
	}
	for name, form := range cases {
		rec := f.post(t, form)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "sk-test-FAKE") || strings.Contains(rec.Body.String(), strings.Repeat("a", 100)) {
			t.Errorf("%s: the refusal echoes the key", name)
		}
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("invalid input stored a row")
	}
	// Switching provider without a new key is refused while a key is saved.
	f.post(t, aiForm_("save", testPassword, fakeAIKey))
	v := aiForm_("save", testPassword, "")
	v.Set("provider", "anthropic")
	if rec := f.post(t, v); rec.Code != http.StatusBadRequest {
		t.Errorf("provider switch without a key = %d, want 400", rec.Code)
	}
	if f.h.coach().Name() != "openai" {
		t.Error("a refused switch changed the provider")
	}
	v.Set("api_key", fakeAIKey)
	if rec := f.post(t, v); rec.Code != http.StatusSeeOther || f.h.coach().Name() != "anthropic" {
		t.Errorf("provider switch with a key = %d, provider %v", rec.Code, f.h.coach())
	}
}

func TestAISettingsMethodAndBodyCap(t *testing.T) {
	f := newAIFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/settings/ai", nil)
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /settings/ai = %d, want 405", rec.Code)
	}
	big := aiForm_("save", testPassword, fakeAIKey)
	big.Set("chat_model", strings.Repeat("a", 20<<10))
	if rec := f.post(t, big); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body = %d, want 400", rec.Code)
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("an oversized body stored settings")
	}
}

func TestAISettingsPasswordAttemptsAreThrottled(t *testing.T) {
	f := newAIFixture(t)
	var last int
	for i := 0; i < 8; i++ {
		last = f.post(t, aiForm_("save", "wrong", fakeAIKey)).Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("after repeated failures status = %d, want 429", last)
	}
	// Even the right password is refused while blocked, and nothing is stored.
	if rec := f.post(t, aiForm_("save", testPassword, fakeAIKey)); rec.Code != http.StatusTooManyRequests {
		t.Errorf("right password while blocked = %d, want 429", rec.Code)
	}
	if row, _ := f.h.Service.Repo.GetAISettings(); row != nil {
		t.Error("settings stored while throttled")
	}
}

type envCoach struct{}

func (*envCoach) Name() string { return "fake" }
func (*envCoach) Respond(context.Context, healthcoach.Request) (healthcoach.Response, error) {
	return healthcoach.Response{}, nil
}

// A key saved only in Settings must also enable photo analysis (#217/#218):
// the photo handler resolves its provider through h.coach(), and the
// settings-built OpenAI provider implements FoodPhotoAnalyzer.
func TestAISettingsKeyEnablesPhotoAnalysis(t *testing.T) {
	f := newAIFixture(t)
	if rec := f.post(t, aiForm_("save", testPassword, fakeAIKey)); rec.Code != http.StatusSeeOther {
		t.Fatalf("save = %d", rec.Code)
	}
	if f.h.Coach != nil {
		t.Fatal("fixture unexpectedly has an environment-configured Coach")
	}
	if _, ok := f.h.coach().(healthcoach.FoodPhotoAnalyzer); !ok {
		t.Fatal("the Settings-built provider cannot analyse photos")
	}
}
