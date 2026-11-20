package secretary_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/secretary/secretarytest"
)

// A calendar chunk wires its reader and its Syncer into Deps; the brief then
// shows its events, each title in an untrusted envelope, with the as_of the
// Syncer reported. secretarytest supplies the database, the clock and the fake.
// The slot's Syncer is not called here because its copy is fresh.
func Example() {
	e, closeDB := secretarytest.Demo()
	defer closeDB()
	fresh := e.Svc.Now().Add(-5 * time.Minute)
	cat, _ := secretary.New(secretary.Deps{
		Service:  e.Svc,
		Health:   secretarytest.FakeHealth{},
		Calendar: secretarytest.FakeCalendar{N: 2, SyncedAt: fresh},
		Syncers: map[string]secretary.Syncer{secretary.SourceCalendar: {
			LastSync: func() (time.Time, error) { return fresh, nil },
			Sync:     func(context.Context) error { return nil },
		}},
	})
	out, _ := secretarytest.RunTool(cat, "get_day_brief", `{"date":"today"}`)
	var brief struct {
		Calendar json.RawMessage `json:"calendar"`
	}
	_ = json.Unmarshal([]byte(out), &brief)
	fmt.Println(string(brief.Calendar))
	// Output:
	// {"as_of":"2026-11-19T11:55","connected":true,"events":[{"start":"2026-11-19T08:00","end":"2026-11-19T08:25","busy":true,"title":{"untrusted":true,"source":"calendar","text":"Fictional meeting 1","truncated":false}},{"start":"2026-11-19T08:30","end":"2026-11-19T08:55","busy":false,"title":{"untrusted":true,"source":"calendar","text":"Fictional meeting 2","truncated":false}}],"more":0,"stale":false,"total":2}
}
