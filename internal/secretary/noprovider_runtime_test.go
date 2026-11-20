package secretary_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/secretary/secretarytest"
)

type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return nil, context.Canceled
}

// Every tool, the brief and a Hevy refresh together make zero HTTP requests.
// The refresh runs against a fake provider, so any request here would be the
// Secretary code reaching out by itself.
func TestNoProviderRequestAtRuntime(t *testing.T) {
	counter := &countingTransport{}
	old := http.DefaultTransport
	http.DefaultTransport = counter
	t.Cleanup(func() { http.DefaultTransport = old })

	e := secretarytest.New(t)
	e.SeedTasks()
	e.SeedProjects()
	e.SeedCourses()
	e.SeedHevy(e.Svc.Now().Add(-3 * time.Hour))
	e.ConnectHevy(&secretarytest.FakeHevy{})
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: 5})
	for _, name := range cat.Names() {
		for _, in := range edgeInputs {
			_, _ = secretarytest.Run(t, cat, name, in)
		}
	}
	day, _ := secretary.ResolveDate("today", cat.Clock())
	if _, err := cat.BuildDayBrief(day); err != nil {
		t.Fatal(err)
	}
	cat.Freshness().EnsureFresh(context.Background(), secretary.SourceHevy, secretary.SourceCalendar, secretary.SourceMail)
	cat.Wait()
	if n := counter.n.Load(); n != 0 {
		t.Fatalf("%d HTTP requests", n)
	}
}
