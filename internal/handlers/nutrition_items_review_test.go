package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestParseAmount(t *testing.T) {
	for raw, want := range map[string]float64{"1": 1, " 2.5 ": 2.5, "1,5": 1.5, "0,25": 0.25, "100": 100} {
		if got, ok := parseAmount(raw); !ok || got != want {
			t.Errorf("parseAmount(%q) = %v %v, want %v", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"1,000", "1,000.5", "1.000,5", "1,2,3", ",5", "1,", "1,234", "abc", "", "1e", "1,5x"} {
		if got, ok := parseAmount(raw); ok {
			t.Errorf("parseAmount(%q) = %v, want refused", raw, got)
		}
	}
}

func TestNutritionThousandsSeparatorIs400(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test barley soup", viewFloat(200))
	_, ref, key := itemsSearch(t, h, "barley")
	rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"1,000"}, "slot": {"lunch"}}, nil)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "thousands separators") || diaryCount(t, h) != 0 {
		t.Fatalf("1,000: %d count %d", rec.Code, diaryCount(t, h))
	}
	rec = itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"1,5"}, "slot": {"lunch"}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("1,5: %d", rec.Code)
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].Amount != 1.5 {
		t.Fatalf("1,5 stored as %+v", items)
	}
}

// My foods are logged only through their live row (hf:), never through the
// mirrored catalogue row (nf:).
func TestNutritionNfRefusesMirroredMyFood(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test pumpkin soup", viewFloat(150))
	_, ref, key := itemsSearch(t, h, "pumpkin")
	if rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"1"}, "slot": {"lunch"}}, nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("hf add: %d", rec.Code)
	}
	rows, _ := h.Service.Repo.SearchNutritionFoods("pumpkin", 5)
	if len(rows) != 1 || rows[0].Provider != "user" {
		t.Fatalf("mirror rows: %+v", rows)
	}
	rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {"nf:" + strconv.FormatInt(rows[0].ID, 10)}, "key": {strings.Repeat("c", 32)}, "amount": {"1"}, "slot": {"lunch"}}, nil)
	if rec.Code != http.StatusBadRequest || diaryCount(t, h) != 1 {
		t.Fatalf("nf on a My food mirror: %d count %d", rec.Code, diaryCount(t, h))
	}
	// And search never offers the mirror row.
	if page, _, _ := itemsSearch(t, h, "pumpkin"); strings.Count(page, "Test pumpkin soup") < 1 || strings.Contains(page, `value="nf:`) {
		t.Fatal("search offered the mirrored row")
	}
}

func TestNutritionGystDayWithNoKcalWording(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test plain tea", nil)
	_, ref, key := itemsSearch(t, h, "tea")
	if rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {ref}, "key": {key}, "amount": {"1"}, "slot": {"snack"}}, nil); rec.Code != http.StatusSeeOther {
		t.Fatal(rec.Code)
	}
	page := itemsDo(h, "GET", "/health/nutrition", nil, nil).Body.String()
	if !strings.Contains(page, "kcal unknown for today's items") || strings.Contains(page, "No food logged yet today") {
		t.Fatalf("calories wording: %s", page)
	}
	if !strings.Contains(page, "No value on today's items") || strings.Contains(page, "Not received") {
		t.Fatal("macro wording")
	}
	if today := itemsDo(h, "GET", "/health", nil, nil).Body.String(); strings.Contains(today, "No food logged yet") {
		t.Fatal("Health says no food logged on a GYST day")
	}
}
