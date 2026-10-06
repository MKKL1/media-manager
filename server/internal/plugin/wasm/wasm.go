package wasm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	extism "github.com/extism/go-sdk"

	"server/internal/library"
	"server/internal/plugin"
	"server/internal/view"
)

type Config struct {
	Path         string
	AllowedHosts []string
	Config       map[string]string
}

type Loaded struct {
	Provider plugin.Plugin
	Type     view.Type
}

type Plugin struct {
	compiled *extism.CompiledPlugin
	info     info
}

var (
	_ plugin.Tags = (*Plugin)(nil)
	_ view.Type   = (*Plugin)(nil)
)

type info struct {
	Provider     library.Provider `json:"provider"`
	Arrangements []string         `json:"arrangements"`
	Tags         []plugin.Tag     `json:"tags"`
	CanSearch    bool             `json:"can_search"`
	Formats      []plugin.Tag     `json:"formats"`
	Fields       []library.Field  `json:"fields"`
}

type entry struct {
	ID             string                   `json:"id"`
	CanHoldEntries bool                     `json:"can_hold_entries"`
	Values         map[library.Field]string `json:"values"`
	Tags           []plugin.Tag             `json:"tags"`
	Entries        []entry                  `json:"entries"`
}

type searchResult struct {
	Ref    string                   `json:"ref"`
	Values map[library.Field]string `json:"values"`
	Tags   []plugin.Tag             `json:"tags"`
}

var limits = &extism.ManifestMemory{MaxPages: 4096, MaxHttpResponseBytes: 50 << 20, MaxVarBytes: 1 << 20}

func Load(ctx context.Context, c Config) (Loaded, error) {
	manifest := extism.Manifest{
		Wasm:         []extism.Wasm{extism.WasmFile{Path: c.Path}},
		Memory:       limits,
		Config:       c.Config,
		AllowedHosts: c.AllowedHosts,
	}
	compiled, err := extism.NewCompiledPlugin(ctx, manifest, extism.PluginConfig{EnableWasi: true}, nil)
	if err != nil {
		return Loaded{}, fmt.Errorf("plugin %s: %w", c.Path, err)
	}
	p := &Plugin{compiled: compiled}
	if err := p.call(ctx, "info", struct{}{}, &p.info); err != nil {
		return Loaded{}, fmt.Errorf("plugin %s: info: %w", c.Path, err)
	}
	if err := p.info.check(); err != nil {
		return Loaded{}, fmt.Errorf("plugin %s: %w", c.Path, err)
	}
	var loaded Loaded
	if len(p.info.Formats) > 0 {
		loaded.Type = p
	}
	if p.info.Provider != "" {
		if len(p.info.Arrangements) == 0 {
			p.info.Arrangements = []string{plugin.Default}
		}
		loaded.Provider = p
		if p.info.CanSearch {
			loaded.Provider = searching{p}
		}
	}
	return loaded, nil
}

var (
	providerName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	tagName      = regexp.MustCompile(`^([a-z0-9-]+):[a-z0-9-]+$`)
)

func (i info) check() error {
	for _, format := range i.Formats {
		if !tagName.MatchString(string(format)) {
			return fmt.Errorf("format %q is not <namespace>:<name>", format)
		}
	}
	if i.Provider == "" && len(i.Formats) > 0 {
		return nil
	}
	if !providerName.MatchString(string(i.Provider)) {
		return fmt.Errorf("provider %q is not lowercase letters, digits and dashes", i.Provider)
	}
	for _, tag := range i.Tags {
		m := tagName.FindStringSubmatch(string(tag))
		if m == nil {
			return fmt.Errorf("tag %q is not <namespace>:<name>", tag)
		}
		if m[1] != "default" && m[1] != string(i.Provider) {
			return fmt.Errorf("tag %q: a plugin tags in default or its own provider (%s) only", tag, i.Provider)
		}
	}
	return nil
}

func (p *Plugin) call(ctx context.Context, fn string, in, out any) error {
	instance, err := p.compiled.Instance(ctx, extism.PluginInstanceConfig{})
	if err != nil {
		return err
	}
	defer instance.Close(ctx)
	input, err := json.Marshal(in)
	if err != nil {
		return err
	}
	_, output, err := instance.CallWithContext(ctx, fn, input)
	if err != nil {
		msg := err.Error()
		if rest, ok := strings.CutPrefix(msg, "not found"); ok {
			return fmt.Errorf("%w%s", plugin.ErrNotFound, rest)
		}
		if rest, ok := strings.CutPrefix(msg, "arrangement not offered"); ok {
			return fmt.Errorf("%w%s", plugin.ErrNoSuchArrangement, rest)
		}
		return err
	}
	return json.Unmarshal(output, out)
}

func (p *Plugin) Provider() library.Provider { return p.info.Provider }
func (p *Plugin) Arrangements() []string     { return p.info.Arrangements }
func (p *Plugin) Tags() []plugin.Tag         { return p.info.Tags }

func (p *Plugin) Fetch(ctx context.Context, arrangement, ref string) (library.Tree, error) {
	var top entry
	if err := p.call(ctx, "fetch", map[string]string{"arrangement": arrangement, "ref": ref}, &top); err != nil {
		return library.Tree{}, err
	}
	return p.fetched(top)
}

func (p *Plugin) fetched(e entry) (library.Tree, error) {
	if e.ID == "" {
		return library.Tree{}, errors.New("an entry has no id")
	}
	values, err := p.withTags(e.Values, e.Tags)
	if err != nil {
		return library.Tree{}, fmt.Errorf("%s: %w", e.ID, err)
	}
	f := library.Tree{ProviderID: e.ID, CanHoldEntries: e.CanHoldEntries, Values: values}
	for _, child := range e.Entries {
		c, err := p.fetched(child)
		if err != nil {
			return library.Tree{}, err
		}
		f.Entries = append(f.Entries, c)
	}
	return f, nil
}

func (p *Plugin) withTags(values map[library.Field]string, tags []plugin.Tag) (map[library.Field]string, error) {
	if _, ok := values[plugin.TagsField]; ok {
		return nil, fmt.Errorf("the %q value comes from tags", plugin.TagsField)
	}
	for _, tag := range tags {
		if !slices.Contains(p.info.Tags, tag) {
			return nil, fmt.Errorf("tag %q is not one the plugin declared", tag)
		}
	}
	if len(tags) > 0 {
		if values == nil {
			values = map[library.Field]string{}
		}
		text := make([]string, len(tags))
		for i, tag := range tags {
			text[i] = string(tag)
		}
		values[plugin.TagsField] = strings.Join(text, " ")
	}
	return values, nil
}

func (p *Plugin) Formats() []plugin.Tag       { return p.info.Formats }
func (p *Plugin) FieldsRead() []library.Field { return p.info.Fields }

func (p *Plugin) Names(ctx context.Context, as plugin.Tag, work view.Entry) (map[library.ID]string, error) {
	if !slices.Contains(p.info.Formats, as) {
		return nil, fmt.Errorf("%w: %s", view.ErrUnknownFormat, as)
	}
	var names map[library.ID]string
	err := p.call(ctx, "format", map[string]any{"format": as, "work": work}, &names)
	return names, err
}

type searching struct{ *Plugin }

func (p searching) Search(ctx context.Context, query string) ([]plugin.SearchResult, error) {
	var found []searchResult
	if err := p.call(ctx, "search", map[string]string{"query": query}, &found); err != nil {
		return nil, err
	}
	results := make([]plugin.SearchResult, len(found))
	for i, f := range found {
		values, err := p.withTags(f.Values, f.Tags)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Ref, err)
		}
		results[i] = plugin.SearchResult{Ref: f.Ref, Values: values}
	}
	return results, nil
}
