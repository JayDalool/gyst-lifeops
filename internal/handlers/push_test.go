package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPushSubscriptionLifecycle(t *testing.T) {
	h := testHandler(t)
	body := `{"endpoint":"https://push.example/subscription","keys":{"p256dh":"public-key","auth":"auth-key"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("subscribe = %d: %s", recorder.Code, recorder.Body.String())
	}
	subscriptions, err := h.Service.Repo.PushSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 || subscriptions[0].Endpoint != "https://push.example/subscription" {
		t.Fatalf("subscriptions = %+v", subscriptions)
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/push/subscribe", strings.NewReader(body))
	recorder = httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unsubscribe = %d", recorder.Code)
	}
}

func TestPushRejectsInsecureEndpoint(t *testing.T) {
	h := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", strings.NewReader(`{"endpoint":"http://push.example","keys":{"p256dh":"x","auth":"y"}}`))
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("insecure endpoint = %d", recorder.Code)
	}
}
