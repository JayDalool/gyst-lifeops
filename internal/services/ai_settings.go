package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/repository"
)

// AI provider settings (GYST-169, Issue #218). The API key is write-only: it
// is stored as AES-256-GCM ciphertext and nonce in ai_settings, the 32-byte
// data key lives in a 0600 file beside the database, and nothing here ever
// returns, logs or formats the key except AIEffective, which hands it to the
// provider client and nowhere else.

const (
	AIProviderOpenAI    = "openai"
	AIProviderAnthropic = "anthropic"

	aiDataKeyBytes = 32
	aiKeyMinLen    = 20
	aiKeyMaxLen    = 300
	aiModelMaxLen  = 64
	aiKeyAADPrefix = "gyst:ai-key:v1:"
)

var (
	aiModelPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	// Length and charset only; no network call. Covers sk-..., sk-proj-...,
	// sk-ant-... keys (letters, digits, "-" and "_").
	aiKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

	// ErrAIKeyFile means the data-key file is missing, unreadable, malformed or
	// too open, or couldn't be created. The text is fixed: it never names a path.
	ErrAIKeyFile = userError("The encryption key file next to the database can't be used, so nothing was saved.")
	// ErrAIStore means the settings couldn't be written. Fixed text.
	ErrAIStore = userError("The AI settings couldn't be saved.")
)

// AISettingsInput is a save request. APIKey may be empty to keep the saved
// key (same provider only).
type AISettingsInput struct {
	Provider   string
	ChatModel  string
	PhotoModel string
	APIKey     string
}

// AISettingsView is what the Settings page may show. It has no key.
type AISettingsView struct {
	Provider      string
	ChatModel     string
	PhotoModel    string
	KeySaved      bool   // a key is stored and decrypts
	KeySavedOn    string // local date, "" when none
	KeyUnreadable bool   // ciphertext is stored but can't be decrypted
}

// AIConfig is the effective configuration saved in settings. Key is secret:
// pass it to a provider client and never render, log or store it.
type AIConfig struct {
	Provider   string
	Key        string
	ChatModel  string
	PhotoModel string
}

func (s *Service) dataKeyPath() string { return s.AIKeyPath }

// loadDataKey reads the data key. A missing file is os.ErrNotExist; anything
// else wrong with it is ErrAIKeyFile.
func (s *Service) loadDataKey() ([]byte, error) {
	path := s.dataKeyPath()
	if path == "" {
		return nil, ErrAIKeyFile
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, ErrAIKeyFile
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != aiDataKeyBytes {
		return nil, ErrAIKeyFile
	}
	// Windows has no Unix mode bits to check; elsewhere group and other access
	// make the key file untrustworthy.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, ErrAIKeyFile
	}
	key := make([]byte, aiDataKeyBytes)
	if _, err := io.ReadFull(f, key); err != nil {
		return nil, ErrAIKeyFile
	}
	return key, nil
}

// ensureDataKey returns the data key, creating the file (mode 0600, O_EXCL)
// on first save.
func (s *Service) ensureDataKey() ([]byte, error) {
	key, err := s.loadDataKey()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	path := s.dataKeyPath()
	fresh := make([]byte, aiDataKeyBytes)
	if _, err := rand.Read(fresh); err != nil {
		return nil, ErrAIKeyFile
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return s.loadDataKey() // lost a race with another process
		}
		return nil, ErrAIKeyFile
	}
	_, werr := f.Write(fresh)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(path)
		return nil, ErrAIKeyFile
	}
	return fresh, nil
}

func aiAEAD(dataKey []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func aiAAD(provider string) []byte { return []byte(aiKeyAADPrefix + provider) }

func encryptAIKey(dataKey []byte, provider, apiKey string) (ciphertext, nonce []byte, err error) {
	aead, err := aiAEAD(dataKey)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return aead.Seal(nil, nonce, []byte(apiKey), aiAAD(provider)), nonce, nil
}

// decryptAIKey fails on any tampering with the ciphertext, nonce or provider.
func decryptAIKey(dataKey []byte, provider string, ciphertext, nonce []byte) (string, error) {
	aead, err := aiAEAD(dataKey)
	if err != nil {
		return "", err
	}
	if len(nonce) != aead.NonceSize() {
		return "", errors.New("bad nonce")
	}
	plain, err := aead.Open(nil, nonce, ciphertext, aiAAD(provider))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// AISettings returns the saved settings for display. It never returns the key.
func (s *Service) AISettings() (AISettingsView, error) {
	s.aiMu.Lock()
	defer s.aiMu.Unlock()
	row, err := s.Repo.GetAISettings()
	if err != nil {
		return AISettingsView{}, ErrAIStore
	}
	if row == nil {
		return AISettingsView{Provider: AIProviderOpenAI}, nil
	}
	view := AISettingsView{Provider: row.Provider, ChatModel: row.ChatModel, PhotoModel: row.PhotoModel}
	if row.KeyCiphertext == nil {
		return view, nil
	}
	if _, ok := s.decryptRow(row); ok {
		view.KeySaved = true
		view.KeySavedOn = s.keySavedDate(row.KeySavedAt)
	} else {
		view.KeyUnreadable = true
	}
	return view, nil
}

func (s *Service) keySavedDate(stored string) string {
	t, err := clock.ParseInstant(stored)
	if err != nil {
		return ""
	}
	if s.Loc != nil {
		t = t.In(s.Loc)
	}
	return t.Format("Jan 2, 2006")
}

// decryptRow returns the key in row, or ok=false for a missing key file, a
// tampered ciphertext or any other failure. Callers treat that as not
// configured and show no detail.
func (s *Service) decryptRow(row *repository.AISettingsRow) (string, bool) {
	if row == nil || row.KeyCiphertext == nil {
		return "", false
	}
	dataKey, err := s.loadDataKey()
	if err != nil {
		return "", false
	}
	plain, err := decryptAIKey(dataKey, row.Provider, row.KeyCiphertext, row.KeyNonce)
	if err != nil || plain == "" {
		return "", false
	}
	return plain, true
}

// AIEffective returns the saved configuration with its decrypted key, or
// ok=false when no usable key is saved (none, tampered, or no key file). It
// is read on every request, so a save or remove takes effect at once.
func (s *Service) AIEffective() (AIConfig, bool) {
	s.aiMu.Lock()
	defer s.aiMu.Unlock()
	row, err := s.Repo.GetAISettings()
	if err != nil || row == nil {
		return AIConfig{}, false
	}
	key, ok := s.decryptRow(row)
	if !ok {
		return AIConfig{}, false
	}
	return AIConfig{Provider: row.Provider, Key: key, ChatModel: row.ChatModel, PhotoModel: row.PhotoModel}, true
}

func validateAIModel(name string) error {
	if name != "" && !aiModelPattern.MatchString(name) {
		return userError("Model names use letters, digits, dots, dashes and underscores, up to 64 characters.")
	}
	return nil
}

// SaveAISettings validates and stores the provider, models and (optionally) a
// new key. The caller has already checked the password.
func (s *Service) SaveAISettings(in AISettingsInput) error {
	in.Provider = strings.ToLower(strings.TrimSpace(in.Provider))
	in.ChatModel = strings.TrimSpace(in.ChatModel)
	in.PhotoModel = strings.TrimSpace(in.PhotoModel)
	in.APIKey = strings.TrimSpace(in.APIKey)
	if in.Provider != AIProviderOpenAI && in.Provider != AIProviderAnthropic {
		return userError("Choose OpenAI or Anthropic.")
	}
	if err := validateAIModel(in.ChatModel); err != nil {
		return err
	}
	if err := validateAIModel(in.PhotoModel); err != nil {
		return err
	}
	if in.APIKey != "" && (len(in.APIKey) < aiKeyMinLen || len(in.APIKey) > aiKeyMaxLen || !aiKeyPattern.MatchString(in.APIKey)) {
		return userError("That doesn't look like an API key. Keys are 20 to 300 letters, digits, dashes or underscores.")
	}

	s.aiMu.Lock()
	defer s.aiMu.Unlock()
	existing, err := s.Repo.GetAISettings()
	if err != nil {
		return ErrAIStore
	}
	row := repository.AISettingsRow{
		Provider: in.Provider, ChatModel: in.ChatModel, PhotoModel: in.PhotoModel,
		UpdatedAt: clock.FormatInstant(s.Now()),
	}
	switch {
	case in.APIKey != "":
		dataKey, err := s.ensureDataKey()
		if err != nil {
			return err
		}
		ct, nonce, err := encryptAIKey(dataKey, in.Provider, in.APIKey)
		if err != nil {
			return ErrAIStore
		}
		row.KeyCiphertext, row.KeyNonce, row.KeySavedAt = ct, nonce, row.UpdatedAt
	case existing != nil && existing.KeyCiphertext != nil:
		if existing.Provider != in.Provider {
			return userError("Enter the API key for the new provider.")
		}
		row.KeyCiphertext, row.KeyNonce, row.KeySavedAt = existing.KeyCiphertext, existing.KeyNonce, existing.KeySavedAt
	}
	if err := s.Repo.SaveAISettings(row); err != nil {
		return ErrAIStore
	}
	return nil
}

// RemoveAIKey deletes the saved key and keeps the provider and models.
func (s *Service) RemoveAIKey() error {
	s.aiMu.Lock()
	defer s.aiMu.Unlock()
	if _, err := s.Repo.ClearAIKey(clock.FormatInstant(s.Now())); err != nil {
		return ErrAIStore
	}
	return nil
}
