// Package hevy implements the read-only public Hevy API.
package hevy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lifeops-tracker/internal/models"
	"net/http"
	"time"
)

type Client struct {
	key, base string
	http      *http.Client
}

func New(key string) *Client {
	return &Client{key: key, base: "https://api.hevyapp.com", http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type template struct {
	ID        string
	Primary   string   `json:"primary_muscle_group"`
	Secondary []string `json:"secondary_muscle_groups"`
}
type entry struct {
	ID, Title string
	Start     string `json:"start_time"`
	Exercises []struct {
		Title    string
		Template string `json:"exercise_template_id"`
		Sets     []struct {
			Type     string
			Weight   *float64 `json:"weight_kg"`
			Reps     *float64
			Range    *struct{ Start, End *float64 } `json:"rep_range"`
			Distance *float64                       `json:"distance_meters"`
			Duration *float64                       `json:"duration_seconds"`
			RPE      *float64
		}
	}
}

func (c *Client) pages(ctx context.Context, path, field string, size int, consume func(json.RawMessage) error) error {
	for page := 1; page <= 1000; page++ {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/%s?page=%d&pageSize=%d", c.base, path, page, size), nil)
		if err != nil {
			return errors.New("Hevy request could not be created")
		}
		req.Header.Set("api-key", c.key)
		res, err := c.http.Do(req)
		if err != nil {
			return errors.New("Hevy could not be reached; cached data retained")
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 4*1024*1024+1))
		res.Body.Close()
		if res.StatusCode == 401 || res.StatusCode == 403 {
			return errors.New("Hevy rejected the API key; check your Pro subscription and key")
		}
		if res.StatusCode == 429 {
			return errors.New("Hevy rate limit reached; try again later")
		}
		if res.StatusCode != 200 || readErr != nil || len(body) > 4*1024*1024 {
			return errors.New("Hevy returned an incomplete response; cached data retained")
		}
		var envelope struct {
			Page  int
			Count int `json:"page_count"`
		}
		var raw map[string]json.RawMessage
		if json.Unmarshal(body, &envelope) != nil || json.Unmarshal(body, &raw) != nil || envelope.Page != page || envelope.Count < 0 || (envelope.Count < page && !(page == 1 && envelope.Count == 0)) || (len(raw[field]) == 0 || raw[field][0] != '[') {
			return errors.New("Hevy returned an invalid page")
		}
		if err := consume(raw[field]); err != nil {
			return errors.New("Hevy returned invalid training data")
		}
		if page >= envelope.Count {
			return nil
		}
	}
	return errors.New("Hevy history exceeds the import limit; cached data retained")
}
func (c *Client) Fetch(ctx context.Context) ([]models.TrainingEntry, error) {
	catalog := map[string]template{}
	err := c.pages(ctx, "exercise_templates", "exercise_templates", 100, func(raw json.RawMessage) error {
		var rows []template
		if err := json.Unmarshal(raw, &rows); err != nil {
			return err
		}
		for _, v := range rows {
			if v.ID == "" {
				return errors.New("missing template ID")
			}
			catalog[v.ID] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := []models.TrainingEntry{}
	for _, kind := range []string{"routine", "workout"} {
		seen := map[string]bool{}
		err = c.pages(ctx, kind+"s", kind+"s", 10, func(raw json.RawMessage) error {
			var rows []entry
			if err := json.Unmarshal(raw, &rows); err != nil {
				return err
			}
			for _, v := range rows {
				if v.ID == "" || seen[v.ID] {
					return errors.New("missing or repeated ID")
				}
				seen[v.ID] = true
				e := models.TrainingEntry{ID: v.ID, Kind: kind, Title: v.Title}
				if kind == "workout" {
					var err error
					e.StartedAt, err = time.Parse(time.RFC3339, v.Start)
					if err != nil {
						return err
					}
				}
				for _, x := range v.Exercises {
					t := catalog[x.Template]
					ex := models.TrainingExercise{Name: x.Title, PrimaryMuscle: t.Primary, SecondaryMuscles: t.Secondary}
					for _, s := range x.Sets {
						set := models.TrainingSet{Kind: s.Type, LoadKG: s.Weight, Reps: s.Reps, DistanceM: s.Distance, Seconds: s.Duration, Effort: s.RPE}
						if s.Range != nil {
							set.MinReps = s.Range.Start
							set.MaxReps = s.Range.End
						}
						ex.Sets = append(ex.Sets, set)
					}
					e.Exercises = append(e.Exercises, ex)
				}
				result = append(result, e)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
