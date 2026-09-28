package clock

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The guard keeps calendar days on the application clock (GYST-VN-003). It
// scans every non-test Go file under internal/ except this package (the one
// sanctioned source) and internal/db (backup timing and migration stamps, no
// calendar logic). It fails on:
//
//   - time.Now, time.Local, time.LoadLocation and time.Unix*;
//   - a zero-argument .Local() call;
//   - a string literal containing the default zone name;
//   - a calendar day taken straight from a Now() call without .In(...):
//     .Format("…2006-01-02…"), .Year(), .Month(), .Day(), .YearDay(),
//     .Weekday(), .Hour(), .Date(), .Truncate(…).
//
// A deliberate instant use can be allowed with a "//gyst:instant <reason>"
// comment on the same line or the line above.

var bannedTimeIdents = map[string]bool{"Now": true, "Local": true, "LoadLocation": true, "Unix": true, "UnixMilli": true, "UnixMicro": true}
var dayMethods = map[string]bool{"Year": true, "Month": true, "Day": true, "YearDay": true, "Weekday": true, "Hour": true, "Date": true, "Truncate": true}

type guardFinding struct {
	pos    token.Position
	reason string
}

func scanSource(t *testing.T, fset *token.FileSet, name string, src any) []guardFinding {
	t.Helper()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	timeName := ""
	for _, imp := range file.Imports {
		if path, _ := strconv.Unquote(imp.Path.Value); path == "time" {
			timeName = "time"
			if imp.Name != nil {
				timeName = imp.Name.Name
			}
		}
	}
	allowed := map[int]bool{}
	var findings []guardFinding
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if !strings.HasPrefix(text, "gyst:instant") {
				continue
			}
			line := fset.Position(comment.Pos()).Line
			if strings.TrimSpace(strings.TrimPrefix(text, "gyst:instant")) == "" {
				findings = append(findings, guardFinding{fset.Position(comment.Pos()), "//gyst:instant needs a reason"})
				continue
			}
			allowed[line] = true
			allowed[line+1] = true
		}
	}
	report := func(node ast.Node, reason string) {
		pos := fset.Position(node.Pos())
		if !allowed[pos.Line] {
			findings = append(findings, guardFinding{pos, reason})
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.SelectorExpr:
			if ident, ok := n.X.(*ast.Ident); ok && timeName != "" && ident.Name == timeName && ident.Obj == nil && bannedTimeIdents[n.Sel.Name] {
				report(n, "time."+n.Sel.Name+" outside internal/clock")
			}
		case *ast.BasicLit:
			if n.Kind == token.STRING && strings.Contains(n.Value, DefaultTimezone) {
				report(n, "hard-coded "+DefaultTimezone+"; use the application clock")
			}
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "Local" && len(n.Args) == 0 {
				report(n, ".Local() uses the process timezone")
			}
			isDay := dayMethods[sel.Sel.Name]
			if sel.Sel.Name == "Format" && len(n.Args) == 1 {
				if lit, ok := n.Args[0].(*ast.BasicLit); ok && strings.Contains(lit.Value, "2006-01-02") {
					isDay = true
				}
			}
			if isDay && chainFromNow(sel.X) {
				report(n, "calendar day taken from Now() without .In(location)")
			}
		}
		return true
	})
	return findings
}

// chainFromNow reports whether expr is a call chain that starts at a Now()
// call and does not pass through .In(...).
func chainFromNow(expr ast.Expr) bool {
	for {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		switch sel.Sel.Name {
		case "Now":
			return true
		case "In":
			return false
		}
		expr = sel.X
	}
}

func TestNoRawClockOutsideClockPackage(t *testing.T) {
	fset := token.NewFileSet()
	var findings []guardFinding
	root := filepath.Join("..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if base := entry.Name(); path != root && (base == "clock" || base == "db") && filepath.Dir(path) == root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		findings = append(findings, scanSource(t, fset, path, nil)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("%s: %s", f.pos, f.reason)
	}
}

func TestGuardFlagsRawClockUse(t *testing.T) {
	bad := map[string]string{
		"now":            "package x\nimport \"time\"\nvar _ = time.Now()",
		"now value":      "package x\nimport \"time\"\nvar f = time.Now",
		"aliased":        "package x\nimport t \"time\"\nvar _ = t.Now()",
		"local":          "package x\nimport \"time\"\nvar _ = time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)",
		"local method":   "package x\nfunc f(x interface{ Local() int }) int { return x.Local() }",
		"load location":  "package x\nimport \"time\"\nvar _, _ = time.LoadLocation(\"Asia/Tokyo\")",
		"unix":           "package x\nimport \"time\"\nvar _ = time.Unix(0, 0)",
		"zone literal":   "package x\nconst h = \"X-WR-TIMEZONE:America/Winnipeg\"",
		"format on now":  "package x\nfunc f(s interface{ Now() T }) string { return s.Now().Format(\"2006-01-02\") }\ntype T interface{ Format(string) string }",
		"day via utc":    "package x\nimport \"time\"\nfunc f(s interface{ Now() time.Time }) int { return s.Now().UTC().Day() }",
		"hour on nested": "package x\nimport \"time\"\ntype S struct{ Service interface{ Now() time.Time } }\nfunc f(h S) int { return h.Service.Now().Hour() }",
		"empty reason":   "package x\nimport \"time\"\nvar _ = time.Now() //gyst:instant",
	}
	for name, src := range bad {
		if findings := scanSource(t, token.NewFileSet(), name+".go", src); len(findings) == 0 {
			t.Errorf("%s: not flagged:\n%s", name, src)
		}
	}
}

func TestGuardAllowsInstantsAndLocalDays(t *testing.T) {
	good := map[string]string{
		"comment":        "package x\n// time.Now() in a comment\nvar _ = 1",
		"string":         "package x\nconst s = \"time.Now()\"",
		"instant":        "package x\nimport \"time\"\nfunc f(s interface{ Now() time.Time }) time.Time { return s.Now() }",
		"in location":    "package x\nimport \"time\"\nfunc f(s interface{ Now() time.Time }, l *time.Location) int { return s.Now().In(l).Hour() }",
		"utc rfc3339":    "package x\nimport \"time\"\nfunc f(s interface{ Now() time.Time }) string { return s.Now().UTC().Format(time.RFC3339Nano) }",
		"annotated":      "package x\nimport \"time\"\nvar _ = time.Now() //gyst:instant ICS stamp",
		"annotated prev": "package x\nimport \"time\"\n//gyst:instant backup file name\nvar _ = time.Now()",
		"shadowed time":  "package x\ntype T struct{ Now func() int }\nfunc f(time T) int { return time.Now() }",
		"utc date":       "package x\nimport \"time\"\nvar _ = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)",
	}
	for name, src := range good {
		if findings := scanSource(t, token.NewFileSet(), name+".go", src); len(findings) != 0 {
			t.Errorf("%s: flagged %v:\n%s", name, findings, src)
		}
	}
}
