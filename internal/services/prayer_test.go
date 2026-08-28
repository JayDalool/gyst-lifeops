package services

import (
	"testing"
	"time"
)

func winnipegTime(t *testing.T, value string) time.Time {
	t.Helper()
	location, err := time.LoadLocation(defaultPrayerTimezone)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, location)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestPrayerDefaultsAndKnownWinnipegDates(t *testing.T) {
	service := testService(t)
	config, err := service.PrayerConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Latitude != 49.8951 || config.Longitude != -97.1384 || config.Timezone != "America/Winnipeg" || config.Method != PrayerMethodISNA || config.AsrMethod != AsrMethodShafi {
		t.Fatalf("unexpected defaults: %#v", config)
	}

	tests := []struct {
		name     string
		date     string
		expected []string
	}{
		{"summer solstice", "2026-06-21 12:00", []string{"02:41", "13:30", "17:50", "21:41", "00:20"}},
		{"spring equinox", "2026-03-20 12:00", []string{"06:01", "13:36", "16:55", "19:41", "21:11"}},
		{"DST boundary", "2026-03-08 12:00", []string{"06:28", "13:39", "16:43", "19:22", "20:50"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := service.PrayerTimes(winnipegTime(t, tt.date))
			if err != nil {
				t.Fatal(err)
			}
			if len(schedule.Times) != 5 {
				t.Fatalf("got %d prayer times", len(schedule.Times))
			}
			for i, expected := range tt.expected {
				if got := schedule.Times[i].At.Format("15:04"); got != expected {
					t.Errorf("%s = %s, want %s", schedule.Times[i].Name, got, expected)
				}
			}
		})
	}
}

func TestPrayerSettingsMethodsAdjustmentsAndHanafiAsr(t *testing.T) {
	service := testService(t)
	date := winnipegTime(t, "2026-03-20 12:00")
	standard, err := service.PrayerTimes(date)
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{
		"prayer.method":       PrayerMethodMWL,
		"prayer.asr_method":   AsrMethodHanafi,
		"prayer.adjust.fajr":  "3",
		"prayer.adjust.dhuhr": "-2",
	}
	for key, value := range settings {
		if err := service.Repo.SetSetting(key, value); err != nil {
			t.Fatal(err)
		}
	}
	adjusted, err := service.PrayerTimes(date)
	if err != nil {
		t.Fatal(err)
	}
	if !adjusted.Times[0].At.Before(standard.Times[0].At) {
		t.Error("MWL Fajr should remain earlier than ISNA after a +3 minute adjustment")
	}
	if got := adjusted.Times[1].At.Sub(standard.Times[1].At); got != -2*time.Minute {
		t.Fatalf("Dhuhr adjustment = %v, want -2m", got)
	}
	if !adjusted.Times[2].At.After(standard.Times[2].At) {
		t.Error("Hanafi Asr should be later than Shafi Asr")
	}
}

func TestNextPrayerAndTomorrowRollover(t *testing.T) {
	service := testService(t)
	now := winnipegTime(t, "2026-03-20 14:00")
	next, err := service.NextPrayer(now)
	if err != nil {
		t.Fatal(err)
	}
	if next.Prayer.Name != "Asr" || next.Remaining <= 0 || next.Tomorrow {
		t.Fatalf("afternoon next prayer: %#v", next)
	}
	afterIsha := winnipegTime(t, "2026-03-20 23:30")
	next, err = service.NextPrayer(afterIsha)
	if err != nil {
		t.Fatal(err)
	}
	if next.Prayer.Name != "Fajr" || !next.Tomorrow || next.Prayer.At.Day() != 21 || next.Remaining <= 0 {
		t.Fatalf("after-Isha rollover: %#v", next)
	}
}
