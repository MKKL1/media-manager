package internal_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

var allowed = map[string][]string{
	"server/internal/library":     {"server/internal/event"},
	"server/internal/event":       {},
	"server/internal/user":        {"server/internal/event"},
	"server/internal/plugin":      {"server/internal/library"},
	"server/internal/plugin/wasm": {"server/internal/library", "server/internal/plugin", "server/internal/view", "github.com/extism/go-sdk"},
	"server/internal/view":        {"server/internal/library", "server/internal/plugin"},

	"server/internal/library/postgres": {"server/internal/library", "server/internal/library/postgres/queries", "github.com/jackc/pgx/v5", "github.com/jackc/pgx/v5/pgxpool"},
	"server/internal/user/postgres":    {"server/internal/user", "server/internal/user/postgres/queries", "github.com/jackc/pgx/v5", "github.com/jackc/pgx/v5/pgxpool"},
}

func TestModulesImportOnlyWhatTheyMay(t *testing.T) {
	for pkg, may := range allowed {
		out, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, pkg).Output()
		if err != nil {
			t.Fatalf("go list %s: %v", pkg, err)
		}
		for imp := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {

			if first, _, _ := strings.Cut(imp, "/"); imp != "" && (strings.Contains(first, ".") || first == "server") && !slices.Contains(may, imp) {
				t.Errorf("%s imports %s", pkg, imp)
			}
		}
	}
}

func TestOnlyAppImportsAnAdapter(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{.ImportPath}} {{join .Imports " "}}`, "server/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		pkg, imports, _ := strings.Cut(line, " ")
		for imp := range strings.FieldsSeq(imports) {
			if strings.HasSuffix(imp, "/postgres") && pkg != "server/app" {
				t.Errorf("%s imports %s", pkg, imp)
			}
		}
	}
}
