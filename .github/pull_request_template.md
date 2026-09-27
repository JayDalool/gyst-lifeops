Refs #<!-- issue number (use Refs, not Closes/Fixes) -->

## Summary
<!-- What changed and why, in 2–4 lines. GYST-VN ID if any. -->

## Files / areas changed
<!-- Main files or areas. Note any hotspot files (handlers.go, middleware.go, shell, galaxy*, migrations). -->

## Tests
<!-- Commands actually run and their result. -->
- [ ] `gofmt -l .` clean · `go vet ./...` · `go build ./...`
- [ ] `go test ./...` · `go test -race ./...`
- [ ] Other checks named in the Issue (e.g. `node scripts/check-ui-shell.cjs`, `node scripts/check-galaxy.cjs`)

## Browser / device verification
<!-- UI only: phone 390×844 and desktop 1280×800, light and dark; real iPhone/PWA if required. Write "not verified" where true. Galaxy: 17A §L checklist. Otherwise "n/a". -->

## Acceptance criteria
<!-- Each criterion from the Issue: PASS / FAIL / PARTIAL / NOT VERIFIED, with evidence. -->
| Criterion | Result | Evidence |
|---|---|---|

## Security impact
<!-- None / describe (auth, CSRF, secrets, input handling, host access). Add `needs:security-review` for PRIVILEGED work. -->

## Database / migration impact
<!-- None / new additive migration (name) tested on a backup copy; rollback note. -->

## Known limitations
<!-- What isn't done, not verified, or deferred. -->

## Screenshots
<!-- Optional; fictional/demo data only, never private data. -->
