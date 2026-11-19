package repository

import "database/sql"

// execer is satisfied by *sql.DB and *sql.Tx so multi-write helpers can run
// inside the caller's transaction: the project use cases (GYST-VN-038,
// GYST-085) and the roadmap write plus orphan adoption (GYST-076). One shared
// interface, because two files each declaring their own stopped the merged
// branch from building.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}
