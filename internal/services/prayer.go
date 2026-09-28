package services

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

// SavePrayerConfig validates and persists the user-facing prayer settings.
func (s *Service) SavePrayerConfig(values url.Values) error {
	latitude, err := strconv.ParseFloat(strings.TrimSpace(values.Get("latitude")), 64)
	if err != nil || latitude < -90 || latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(values.Get("longitude")), 64)
	if err != nil || longitude < -180 || longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	// The prayer timezone is the application timezone (GYST_TIMEZONE); a posted
	// "timezone" value is ignored and no longer stored (GYST-VN-003).
	method := strings.ToLower(strings.TrimSpace(values.Get("method")))
	if _, ok := prayerMethods[method]; !ok {
		return fmt.Errorf("invalid prayer method")
	}
	asr := strings.ToLower(strings.TrimSpace(values.Get("asr_method")))
	if asr != AsrMethodShafi && asr != AsrMethodHanafi {
		return fmt.Errorf("invalid Asr method")
	}
	alertSound := strings.ToLower(strings.TrimSpace(values.Get("alert_sound")))
	if alertSound != "chime" && alertSound != "athan" {
		return fmt.Errorf("invalid alert sound")
	}
	athanFile := strings.TrimSpace(values.Get("athan_file"))
	if strings.Contains(athanFile, "/") || strings.Contains(athanFile, "\\") || strings.Contains(athanFile, "..") {
		return fmt.Errorf("Athan file must be a filename from web/static/audio")
	}
	settings := map[string]string{
		"prayer.latitude":  strconv.FormatFloat(latitude, 'f', -1, 64),
		"prayer.longitude": strconv.FormatFloat(longitude, 'f', -1, 64),
		"prayer.method":    method, "prayer.asr_method": asr,
		"prayer.alert_sound": alertSound, "prayer.athan_file": athanFile,
	}
	for _, name := range prayerNames {
		value := strings.TrimSpace(values.Get("adjust_" + name))
		minutes, parseErr := strconv.Atoi(value)
		if parseErr != nil || minutes < -120 || minutes > 120 {
			return fmt.Errorf("%s adjustment must be between -120 and 120 minutes", titlePrayer(name))
		}
		settings["prayer.adjust."+name] = strconv.Itoa(minutes)
	}
	for key, value := range settings {
		if err := s.Repo.SetSetting(key, value); err != nil {
			return err
		}
	}
	return nil
}

const (
	PrayerMethodISNA       = "isna"
	PrayerMethodMWL        = "mwl"
	PrayerMethodUmmAlQura  = "umm_al_qura"
	PrayerMethodEgyptian   = "egyptian"
	PrayerMethodKarachi    = "karachi"
	AsrMethodShafi         = "shafi"
	AsrMethodHanafi        = "hanafi"
	defaultPrayerLatitude  = 49.8951
	defaultPrayerLongitude = -97.1384
)

// defaultPrayerTimezone is the application default; the prayer timezone always
// follows the application clock.
const defaultPrayerTimezone = clock.DefaultTimezone

var prayerNames = []string{"fajr", "dhuhr", "asr", "maghrib", "isha"}

type prayerAngles struct {
	fajr        float64
	isha        float64
	ishaMinutes int
}

var prayerMethods = map[string]prayerAngles{
	PrayerMethodISNA:      {fajr: 15, isha: 15},
	PrayerMethodMWL:       {fajr: 18, isha: 17},
	PrayerMethodUmmAlQura: {fajr: 18.5, ishaMinutes: 90},
	PrayerMethodEgyptian:  {fajr: 19.5, isha: 17.5},
	PrayerMethodKarachi:   {fajr: 18, isha: 18},
}

func (s *Service) PrayerConfig() (models.PrayerConfig, error) {
	config := models.PrayerConfig{
		Latitude:    defaultPrayerLatitude,
		Longitude:   defaultPrayerLongitude,
		Timezone:    s.Location().String(),
		Method:      PrayerMethodISNA,
		AsrMethod:   AsrMethodShafi,
		Adjustments: make(map[string]int, len(prayerNames)),
		AlertSound:  "athan",
	}
	settings := []struct {
		key string
		set func(string)
	}{
		{"prayer.latitude", func(value string) { config.Latitude = floatSetting(value, config.Latitude, -90, 90) }},
		{"prayer.longitude", func(value string) { config.Longitude = floatSetting(value, config.Longitude, -180, 180) }},
		{"prayer.method", func(value string) {
			value = strings.ToLower(strings.TrimSpace(value))
			if _, ok := prayerMethods[value]; ok {
				config.Method = value
			}
		}},
		{"prayer.asr_method", func(value string) {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == AsrMethodShafi || value == AsrMethodHanafi {
				config.AsrMethod = value
			}
		}},
		{"prayer.alert_sound", func(value string) {
			if value == "chime" || value == "athan" {
				config.AlertSound = value
			}
		}},
		{"prayer.athan_file", func(value string) { config.AthanFile = strings.TrimSpace(value) }},
	}
	for _, setting := range settings {
		value, err := s.Repo.GetSetting(setting.key)
		if err != nil {
			return models.PrayerConfig{}, err
		}
		if value != "" {
			setting.set(value)
		}
	}
	for _, name := range prayerNames {
		value, err := s.Repo.GetSetting("prayer.adjust." + name)
		if err != nil {
			return models.PrayerConfig{}, err
		}
		if minutes, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && minutes >= -120 && minutes <= 120 {
			config.Adjustments[name] = minutes
		}
	}
	return config, nil
}

func floatSetting(value string, fallback, min, max float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < min || parsed > max {
		return fallback
	}
	return parsed
}

// PrayerTimes computes a day's five prayer times locally. The calculations use
// NOAA solar coordinates and conventional method angles; no network or cache is
// involved.
func (s *Service) PrayerTimes(date time.Time) (models.PrayerSchedule, error) {
	config, err := s.PrayerConfig()
	if err != nil {
		return models.PrayerSchedule{}, err
	}
	location := s.Location()
	localDate := date.In(location)
	day := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 12, 0, 0, 0, location)
	jd := julianDay(localDate.Year(), localDate.Month(), localDate.Day())
	declination, equation := solarCoordinates(jd + 0.5)
	noon := 720 - 4*config.Longitude - equation

	angles := prayerMethods[config.Method]
	fajr := noon - hourAngleMinutes(config.Latitude, declination, -angles.fajr)
	sunset := noon + hourAngleMinutes(config.Latitude, declination, -0.833)
	factor := 1.0
	if config.AsrMethod == AsrMethodHanafi {
		factor = 2
	}
	asrAltitude := radToDeg(math.Atan(1 / (factor + math.Tan(degToRad(math.Abs(config.Latitude-declination))))))
	asr := noon + hourAngleMinutes(config.Latitude, declination, asrAltitude)
	isha := noon + hourAngleMinutes(config.Latitude, declination, -angles.isha)
	if angles.ishaMinutes > 0 {
		isha = sunset + float64(angles.ishaMinutes)
	}

	minutes := []float64{fajr, noon, asr, sunset, isha}
	times := make([]models.PrayerTime, 0, len(prayerNames))
	for i, name := range prayerNames {
		at := timeFromUTCMinutes(day, minutes[i]+float64(config.Adjustments[name]), location)
		times = append(times, models.PrayerTime{Name: titlePrayer(name), Time: at.Format("3:04 PM"), At: at})
	}
	return models.PrayerSchedule{Date: day.Format("2006-01-02"), Location: "Winnipeg", Times: times}, nil
}

// NextPrayer returns the next prayer strictly after now. After Isha it rolls to
// tomorrow's Fajr in the configured timezone.
func (s *Service) NextPrayer(now time.Time) (models.NextPrayer, error) {
	schedule, err := s.PrayerTimes(now)
	if err != nil {
		return models.NextPrayer{}, err
	}
	for _, prayer := range schedule.Times {
		if prayer.At.After(now) {
			return models.NextPrayer{Prayer: prayer, Remaining: prayer.At.Sub(now)}, nil
		}
	}
	tomorrow, err := s.PrayerTimes(now.AddDate(0, 0, 1))
	if err != nil {
		return models.NextPrayer{}, err
	}
	prayer := tomorrow.Times[0]
	return models.NextPrayer{Prayer: prayer, Remaining: prayer.At.Sub(now), Tomorrow: true}, nil
}

func julianDay(year int, month time.Month, day int) float64 {
	y, m := year, int(month)
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(day+b) - 1524.5
}

func solarCoordinates(jd float64) (declination, equationMinutes float64) {
	t := (jd - 2451545.0) / 36525
	meanLong := normalizeDegrees(280.46646 + t*(36000.76983+t*0.0003032))
	meanAnomaly := 357.52911 + t*(35999.05029-0.0001537*t)
	eccentricity := 0.016708634 - t*(0.000042037+0.0000001267*t)
	center := math.Sin(degToRad(meanAnomaly))*(1.914602-t*(0.004817+0.000014*t)) +
		math.Sin(degToRad(2*meanAnomaly))*(0.019993-0.000101*t) + math.Sin(degToRad(3*meanAnomaly))*0.000289
	trueLong := meanLong + center
	omega := 125.04 - 1934.136*t
	apparentLong := trueLong - 0.00569 - 0.00478*math.Sin(degToRad(omega))
	obliquity := 23 + (26+(21.448-t*(46.815+t*(0.00059-t*0.001813)))/60)/60
	obliquity += 0.00256 * math.Cos(degToRad(omega))
	declination = radToDeg(math.Asin(math.Sin(degToRad(obliquity)) * math.Sin(degToRad(apparentLong))))
	y := math.Pow(math.Tan(degToRad(obliquity)/2), 2)
	equationMinutes = 4 * radToDeg(y*math.Sin(2*degToRad(meanLong))-
		2*eccentricity*math.Sin(degToRad(meanAnomaly))+
		4*eccentricity*y*math.Sin(degToRad(meanAnomaly))*math.Cos(2*degToRad(meanLong))-
		0.5*y*y*math.Sin(4*degToRad(meanLong))-
		1.25*eccentricity*eccentricity*math.Sin(2*degToRad(meanAnomaly)))
	return declination, equationMinutes
}

func hourAngleMinutes(latitude, declination, altitude float64) float64 {
	numerator := math.Sin(degToRad(altitude)) - math.Sin(degToRad(latitude))*math.Sin(degToRad(declination))
	denominator := math.Cos(degToRad(latitude)) * math.Cos(degToRad(declination))
	cosine := math.Max(-1, math.Min(1, numerator/denominator))
	return radToDeg(math.Acos(cosine)) * 4
}

func timeFromUTCMinutes(day time.Time, minutes float64, location *time.Location) time.Time {
	rounded := int(math.Round(minutes))
	midnightUTC := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	return midnightUTC.Add(time.Duration(rounded) * time.Minute).In(location)
}

func normalizeDegrees(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	return value
}

func degToRad(value float64) float64 { return value * math.Pi / 180 }
func radToDeg(value float64) float64 { return value * 180 / math.Pi }

func titlePrayer(name string) string {
	return strings.ToUpper(name[:1]) + name[1:]
}
