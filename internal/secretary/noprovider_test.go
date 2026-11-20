package secretary

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Issue #236 sends nothing to any AI provider. Two checks keep it so: this AST
// guard on the imports and calls of the package's production files, and the
// runtime check with a counting transport in noprovider_runtime_test.go.

const module = "lifeops-tracker/internal/"

// Imports a production file of internal/secretary may have.
var allowedSecretary = map[string]bool{
	"bytes": true, "context": true, "encoding/json": true, "errors": true, "fmt": true, "log/slog": true, "slices": true,
	"strings": true, "sync": true, "time": true, "unicode": true, "unicode/utf8": true,
	module + "clock": true, module + "healthcoach": true, module + "models": true, module + "services": true,
}

// secretarytest is test support. It may use os and path/filepath to build its
// database template, and the data packages; it may not reach the network, run
// a program, or read the environment.
var allowedSecretarytest = map[string]bool{
	"context": true, "database/sql": true, "encoding/json": true, "fmt": true, "os": true, "path/filepath": true, "runtime": true,
	"sort": true, "strings": true, "sync": true, "testing": true, "time": true,
	module + "clock": true, module + "db": true, module + "healthcoach": true, module + "models": true,
	module + "repository": true, module + "secretary": true, module + "services": true,
}

// Names no production file may use, whatever package they come from: the model
// calls, and process/environment reads.
var bannedSelectors = map[string]bool{
	"Respond": true, "NewAnthropic": true, "NewOpenAI": true, "Getenv": true, "LookupEnv": true, "Environ": true, "ExpandEnv": true,
}

func guard(fset *token.FileSet, name string, src any, allowed map[string]bool) []string {
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	var out []string
	healthcoach := ""
	for _, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if !allowed[path] {
			out = append(out, name+": import "+path+" is not allowed")
		}
		if path == module+"healthcoach" {
			healthcoach = "healthcoach"
			if imp.Name != nil {
				healthcoach = imp.Name.Name
			}
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if bannedSelectors[sel.Sel.Name] {
			out = append(out, name+": "+sel.Sel.Name+" is not allowed")
		}
		// healthcoach is used for its Tool type only: no provider, no Request.
		if id, ok := sel.X.(*ast.Ident); ok && healthcoach != "" && id.Name == healthcoach && sel.Sel.Name != "Tool" {
			out = append(out, name+": healthcoach."+sel.Sel.Name+" (only healthcoach.Tool is allowed)")
		}
		return true
	})
	return out
}

func productionFiles(t *testing.T, dir string) []string {
	t.Helper()
	all, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(all) == 0 {
		t.Fatalf("no Go files in %s: %v", dir, err)
	}
	var out []string
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

func TestNoProviderCall(t *testing.T) {
	fset := token.NewFileSet()
	var bad []string
	for _, f := range productionFiles(t, ".") {
		bad = append(bad, guard(fset, f, nil, allowedSecretary)...)
	}
	for _, f := range productionFiles(t, "secretarytest") {
		bad = append(bad, guard(fset, f, nil, allowedSecretarytest)...)
	}
	if len(bad) > 0 {
		t.Fatalf("the Secretary catalogue must not reach a provider, the network or the environment:\n%s", strings.Join(bad, "\n"))
	}
}

// The guard has to catch what it claims to: each snippet breaks one rule.
func TestNoProviderGuardCatchesViolations(t *testing.T) {
	for name, src := range map[string]string{
		"net/http":        "package x\nimport \"net/http\"\nvar _ = http.Get",
		"os":              "package x\nimport \"os\"\nvar _ = os.Args",
		"os/exec":         "package x\nimport \"os/exec\"\nvar _ = exec.Command",
		"handlers":        "package x\nimport \"lifeops-tracker/internal/handlers\"\nvar _ = handlers.New",
		"healthcoach use": "package x\nimport \"lifeops-tracker/internal/healthcoach\"\nvar _ healthcoach.Provider",
		"Respond":         "package x\nimport \"lifeops-tracker/internal/healthcoach\"\nvar _ healthcoach.Tool\nfunc f(p interface{ Respond() }) { p.Respond() }",
		"getenv":          "package x\nvar _ = os.Getenv",
	} {
		if len(guard(token.NewFileSet(), name+".go", src, allowedSecretary)) == 0 {
			t.Errorf("%s was not caught", name)
		}
	}
	if got := guard(token.NewFileSet(), "ok.go", "package x\nimport \"lifeops-tracker/internal/healthcoach\"\nvar _ healthcoach.Tool", allowedSecretary); len(got) != 0 {
		t.Errorf("healthcoach.Tool refused: %v", got)
	}
	if len(guard(token.NewFileSet(), "exec.go", "package x\nimport \"os/exec\"\nvar _ = exec.Command", allowedSecretarytest)) == 0 {
		t.Error("os/exec allowed in secretarytest")
	}
	if len(guard(token.NewFileSet(), "http.go", "package x\nimport \"net/http\"\nvar _ = http.Get", allowedSecretarytest)) == 0 {
		t.Error("net/http allowed in secretarytest")
	}
}
