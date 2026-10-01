package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"

	"lifeops-tracker/internal/repository"
)

// refusedPage is shown when the rules refuse a completion (an inactive project,
// missing proof). It is self-contained on purpose: the installed app has no
// browser Back button, so the page itself must offer a way out, and it must
// still render if the app's own stylesheet has failed to load.
var refusedPage = template.Must(template.New("refused").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<title>GYST · {{.Title}}</title>
<style>
:root { color-scheme: light dark; --bg:#f8fafc; --card:#ffffff; --ink:#0f172a; --muted:#475569; --line:#e2e8f0; --accent:#2563eb; --on-accent:#ffffff; }
@media (prefers-color-scheme: dark) { :root { --bg:#0f172a; --card:#162033; --ink:#e6ecf5; --muted:#a3b0c2; --line:#26344d; --accent:#7aa7ff; --on-accent:#0b1220; } }
body { margin:0; min-height:100vh; display:grid; place-items:center; background:var(--bg); color:var(--ink); font:16px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif; padding:24px 16px; box-sizing:border-box; }
main { box-sizing:border-box; }
main { width:100%; max-width:420px; background:var(--card); border:1px solid var(--line); border-radius:16px; padding:24px; display:flex; flex-direction:column; gap:12px; }
h1 { margin:0; font-size:1.25rem; line-height:1.3; }
p { margin:0; color:var(--muted); }
nav { display:flex; flex-wrap:wrap; gap:8px; margin-top:8px; }
a { display:inline-flex; align-items:center; justify-content:center; min-height:44px; padding:0 16px; border-radius:10px; font-weight:600; text-decoration:none; border:1px solid var(--line); color:var(--ink); }
a.primary { background:var(--accent); border-color:var(--accent); color:var(--on-accent); }
a:focus-visible { outline:2px solid var(--accent); outline-offset:2px; }
</style>
</head>
<body>
<main role="alert">
<h1>{{.Title}}</h1>
<p>{{.Message}}</p>
<nav><a class="primary" href="{{.Back}}">Go back</a><a href="/today">Today</a></nav>
</main>
</body>
</html>
`))

// refuseCompletion answers a completion the rules refused with a readable page
// and a way back. It reports false for any other error, which the caller still
// treats as a server fault.
func refuseCompletion(w http.ResponseWriter, r *http.Request, err error, fallback string) bool {
	var message string
	switch {
	case errors.Is(err, repository.ErrProjectNotActive):
		message = "This project isn't active, so its missions can't be marked done. Reopen the project to keep working on it."
	case errors.Is(err, repository.ErrProofRequired):
		message = "This needs proof before it can be marked done. Add the proof, then try again."
	default:
		return false
	}
	back := safeReturnTo(r)
	if back == "" {
		back = fallback
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	if err := refusedPage.Execute(w, struct{ Title, Message, Back string }{"Couldn't mark that done", message, back}); err != nil {
		log.Printf("render refused page: %v", err)
	}
	return true
}
