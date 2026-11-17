package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// A GET search never writes; the My food is mirrored by the POST that logs it,
// and a scaled value over the database limits is a 400, not a server error.
func TestNutritionSearchWritesNothingAndHugeAmountIs400(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test mega stew", viewFloat(900))
	_, ref, key := itemsSearch(t, h, "mega")
	if !strings.HasPrefix(ref, "hf:") {
		t.Fatalf("My food ref = %q", ref)
	}
	if rows, _ := h.Service.Repo.SearchNutritionFoods("mega", 5); len(rows) != 0 {
		t.Fatalf("a GET search wrote %d catalogue rows", len(rows))
	}
	rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"99999"}, "slot": {"dinner"}, "q": {"mega"}}, nil)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "too large") {
		t.Fatalf("huge amount: %d", rec.Code)
	}
	if rows, _ := h.Service.Repo.SearchNutritionFoods("mega", 5); len(rows) != 0 || diaryCount(t, h) != 0 {
		t.Fatal("a refused add wrote rows")
	}
	if rec = itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"1"}, "slot": {"dinner"}}, nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("add: %d", rec.Code)
	}
	if rows, _ := h.Service.Repo.SearchNutritionFoods("mega", 5); len(rows) != 1 {
		t.Fatalf("add should mirror once: %d", len(rows))
	}
}
