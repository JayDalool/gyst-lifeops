package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestQuickTaskCreatesAndReturnsToCurrentPage(t *testing.T) {
	h := testHandler(t)
	values := url.Values{
		"title":       {"Ship command palette"},
		"priority":    {"high"},
		"category":    {"coding"},
		"due_date":    {"2026-08-29"},
		"return_to":   {"/galaxy"},
		"repeat_type": {"none"},
		"proof_type":  {"none"},
	}
	req := httptest.NewRequest(http.MethodPost, "/tasks/quick", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/galaxy" {
		t.Fatalf("quick task redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	tasks, err := h.Service.Repo.ListTasks(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Ship command palette" || tasks[0].Priority != "high" || tasks[0].StarsReward != 20 {
		t.Fatalf("quick task not persisted: %#v", tasks)
	}
}

func TestQuickTaskRejectsBlankTitle(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/tasks/quick", strings.NewReader("title=+"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("blank quick task status = %d", rec.Code)
	}
}
