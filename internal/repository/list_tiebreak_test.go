package repository

import (
	"reflect"
	"testing"
)

// GYST-167: created_at has one-second resolution and a roadmap import writes
// many rows in one second. SQLite scans such ties in id order today; these
// tests pin that visible order so it is guaranteed (explicit "id ASC" last in
// each ORDER BY) instead of an accident of the query plan. Fictional values.

func tieIDs(t *testing.T, repo *Repository, query string) []int64 {
	t.Helper()
	rows, err := repo.db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestListGoalsTiesKeepIDOrder(t *testing.T) {
	repo := newTestRepo(t)
	for _, title := range []string{"Goal A", "Goal B", "Goal C"} {
		if _, err := repo.db.Exec(`INSERT INTO goals (title, type, status, target_value, created_at)
			VALUES (?, 'monthly', 'active', 1, '2030-01-01 00:00:00')`, title); err != nil {
			t.Fatal(err)
		}
	}
	want := tieIDs(t, repo, `SELECT id FROM goals ORDER BY id ASC`)
	got, err := repo.ListGoals(false)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, g := range got {
		order = append(order, g.Goal.ID)
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("goal order = %v, want %v", order, want)
	}
}

func TestListTasksTiesKeepIDOrder(t *testing.T) {
	repo := newTestRepo(t)
	for _, title := range []string{"Task A", "Task B", "Task C"} {
		if _, err := repo.db.Exec(`INSERT INTO tasks (title, priority, status, created_at)
			VALUES (?, 'medium', 'pending', '2030-01-01 00:00:00')`, title); err != nil {
			t.Fatal(err)
		}
	}
	want := tieIDs(t, repo, `SELECT id FROM tasks ORDER BY id ASC`)
	got, err := repo.ListTasks(false)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, tk := range got {
		order = append(order, tk.ID)
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("task order = %v, want %v", order, want)
	}
}

func TestListRoadmapProjectsTiesKeepIDOrder(t *testing.T) {
	repo := newTestRepo(t)
	for _, name := range []string{"Fictional Atlas", "Orbit Alpha", "Orbit Beta"} {
		res, err := repo.db.Exec(`INSERT INTO projects (name, project_kind, status, created_at)
			VALUES (?, 'main', 'active', '2030-01-01 00:00:00')`, name)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		if _, err := repo.db.Exec(`INSERT INTO roadmap_project_meta (project_id) VALUES (?)`, id); err != nil {
			t.Fatal(err)
		}
	}
	want := tieIDs(t, repo, `SELECT id FROM projects ORDER BY id ASC`)
	got, err := repo.ListRoadmapProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, p := range got {
		order = append(order, p.Project.Project.ID)
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("roadmap project order = %v, want %v", order, want)
	}
}

func TestListRoadmapWeeklyReviewsTiesKeepIDOrder(t *testing.T) {
	repo := newTestRepo(t)
	for _, n := range []string{"a", "b", "c"} {
		if _, err := repo.db.Exec(`INSERT INTO roadmap_weekly_reviews (week_start, finished, created_at)
			VALUES ('2030-01-07', ?, '2030-01-01 00:00:00')`, n); err != nil {
			t.Fatal(err)
		}
	}
	// Unlike the other lists, this one already returned ties newest id first
	// (it walks idx_roadmap_weekly_reviews_week backwards), so id DESC keeps it.
	want := tieIDs(t, repo, `SELECT id FROM roadmap_weekly_reviews ORDER BY id DESC`)
	got, err := repo.ListRoadmapWeeklyReviews(10)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, r := range got {
		order = append(order, r.ID)
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("weekly review order = %v, want %v", order, want)
	}
}
