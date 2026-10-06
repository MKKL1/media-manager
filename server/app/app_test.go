package app

import (
	"context"
	"strings"
	"testing"

	"server/internal/library"
	"server/internal/plugin"
	"server/internal/plugin/wasm"
)

type fake library.Provider

func (f fake) Provider() library.Provider { return library.Provider(f) }
func (fake) Fetch(context.Context, string, string) (library.Tree, error) {
	return library.Tree{}, nil
}

func TestCombineLetsALoadedPluginReplaceABuiltIn(t *testing.T) {
	p := &plugins{builtIn: []plugin.Plugin{fake("tvdb"), fake("anidb")}}
	providers, _, err := p.combine(map[string]wasm.Loaded{"tvdb": {Provider: fake("tvdb")}, "tmdb": {Provider: fake("tmdb")}})
	if err != nil || len(providers) != 3 {
		t.Fatalf("got %v, %v; want tmdb, tvdb, anidb", providers, err)
	}
	_, _, err = p.combine(map[string]wasm.Loaded{"a": {Provider: fake("tmdb")}, "b": {Provider: fake("tmdb")}})
	if err == nil || !strings.Contains(err.Error(), "already tmdb") {
		t.Fatalf("two plugins of one provider: %v", err)
	}
}
