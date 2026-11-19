package handlers

import (
	"errors"
	"net/http"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Settings → AI provider (GYST-169, Issue #218). The API key is write-only:
// no template, response or log line carries it. Saving or removing it needs
// the GYST password again (step-up, 09 §B.7) because the key spends money.

const (
	aiBodyLimit = 16 << 10

	aiMsgEnvManaged = "The server environment configures the AI provider, so these settings are off."
	aiMsgPassword   = "Incorrect password. Nothing was changed."
	aiMsgNotice     = "AI settings saved."
	aiMsgRemoved    = "API key removed."
)

// aiSettingsData is what settings.html shows for the AI provider section.
type aiSettingsData struct {
	EnvManaged bool
	services.AISettingsView
	Notice string
	Error  string
}

// aiForm carries the values to redraw after a refused submit, plus the
// message. It never has a key or a password.
type aiForm struct {
	Provider, ChatModel, PhotoModel string
	Notice, Error                   string
}

// aiNotice maps the ?ai= redirect marker to fixed text; anything else is "".
func aiNotice(marker string) string {
	switch marker {
	case "saved":
		return aiMsgNotice
	case "removed":
		return aiMsgRemoved
	}
	return ""
}

// coach resolves the Health Coach provider for this request. The environment
// takes precedence; otherwise the saved settings are read each time, so a
// save or remove takes effect without a restart. Nil means not configured.
func (h *Handler) coach() healthcoach.Provider {
	if h.Coach != nil || h.AIEnvManaged || h.Service == nil {
		return h.Coach
	}
	cfg, ok := h.Service.AIEffective()
	if !ok {
		return nil
	}
	h.coachMu.Lock()
	defer h.coachMu.Unlock()
	if h.coachBuilt != nil && h.coachCfg == cfg {
		return h.coachBuilt
	}
	var built healthcoach.Provider
	switch cfg.Provider {
	case services.AIProviderAnthropic:
		built = healthcoach.NewAnthropic(cfg.Key, cfg.ChatModel).WithPhotoModel(cfg.PhotoModel)
	case services.AIProviderOpenAI:
		built = healthcoach.NewOpenAI(cfg.Key, cfg.ChatModel).WithVisionModel(cfg.PhotoModel)
	default:
		return nil
	}
	h.coachCfg, h.coachBuilt = cfg, built
	return built
}

func (h *Handler) renderSettings(w http.ResponseWriter, r *http.Request, status int, form aiForm) {
	token, err := h.Service.CalendarToken()
	if err != nil {
		serverError(w, err)
		return
	}
	prayer, err := h.Service.PrayerConfig()
	if err != nil {
		serverError(w, err)
		return
	}
	view, err := h.Service.AISettings()
	if err != nil {
		serverError(w, err)
		return
	}
	if form.Provider != "" {
		view.Provider, view.ChatModel, view.PhotoModel = form.Provider, form.ChatModel, form.PhotoModel
	}
	ai := aiSettingsData{EnvManaged: h.AIEnvManaged, AISettingsView: view, Notice: form.Notice, Error: form.Error}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	h.render(w, "settings.html", SettingsData{Page: "settings", CalendarToken: token, BaseURL: requestBaseURL(r), Prayer: prayer, AI: ai})
}

// aiAudit writes one settings.ai_key.* row. It has no detail, and a failure
// is logged by action and request id only.
func (h *Handler) aiAudit(r *http.Request, action, outcome string) {
	ingress, _, _ := loginClient(r)
	if err := h.Service.RecordAudit(r.Context(), repository.AuditEvent{
		ActorType: services.AuditActorHuman, ActorRef: services.SessionActorRef(sessionToken(r)),
		Ingress: ingress, Action: action, TargetRef: "settings:ai",
		Outcome: outcome, RequestID: requestIDFrom(r.Context()),
	}); err != nil {
		h.log().Error("audit write failed", "action", action, "request_id", requestIDFrom(r.Context()))
	}
}

func (h *Handler) aiSettingsPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, aiBodyLimit)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	form := aiForm{
		Provider:   r.PostForm.Get("provider"),
		ChatModel:  r.PostForm.Get("chat_model"),
		PhotoModel: r.PostForm.Get("photo_model"),
	}
	if h.AIEnvManaged {
		form.Error = aiMsgEnvManaged
		h.renderSettings(w, r, http.StatusConflict, aiForm{Error: form.Error})
		return
	}

	// Step-up. The login throttle counts these attempts too, so this form
	// can't be used to guess the password faster than the login page allows.
	ingress, key, client := loginClient(r)
	attempt := h.admitLogin(ingress, key, client)
	if attempt == nil {
		h.renderSettings(w, r, http.StatusTooManyRequests, aiForm{Error: throttledMessage})
		return
	}
	defer attempt.Release()
	if err := h.Auth.VerifyPassword(r.PostForm.Get("password")); err != nil {
		if !errors.Is(err, services.ErrInvalidCredentials) {
			serverError(w, err)
			return
		}
		attempt.Fail()
		h.aiAudit(r, services.AuditActionReauthFailed, services.AuditOutcomeFailed)
		form.Error = aiMsgPassword
		h.renderSettings(w, r, http.StatusUnauthorized, form)
		return
	}
	attempt.Succeed()

	action, marker, auditAction := r.PostForm.Get("action"), "saved", services.AuditActionAIKeySet
	var err error
	switch action {
	case "save":
		err = h.Service.SaveAISettings(services.AISettingsInput{
			Provider: form.Provider, ChatModel: form.ChatModel, PhotoModel: form.PhotoModel,
			APIKey: r.PostForm.Get("api_key"),
		})
	case "remove":
		marker, auditAction = "removed", services.AuditActionAIKeyRemove
		err = h.Service.RemoveAIKey()
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}
	if err != nil {
		h.aiAudit(r, auditAction, services.AuditOutcomeFailed)
		if message, ok := services.UserMessage(err); ok {
			form.Error = message
			h.renderSettings(w, r, http.StatusBadRequest, form)
			return
		}
		serverError(w, err)
		return
	}
	h.aiAudit(r, auditAction, services.AuditOutcomeSuccess)
	http.Redirect(w, r, "/settings?ai="+marker+"#ai", http.StatusSeeOther)
}
