// Package protoshellmanifest implements .vibex/prototype-manifest.yaml read/write
// aligned with frontend/src/lib/workbench/prototype-shell-manifest.ts and
// frontend/src/routes/api/workspace/prototype-manifest/+server.ts
package protoshellmanifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const ManifestRel = ".vibex/prototype-manifest.yaml"

// Route mirrors ShellManifestRoute (TS).
type Route struct {
	ID        string `yaml:"id" json:"id"`
	Path      string `yaml:"path" json:"path"`
	Title     string `yaml:"title" json:"title"`
	SpecRef   string `yaml:"specRef" json:"specRef"`
	EntryHTML string `yaml:"entryHtml" json:"entryHtml"`
	Kind      string `yaml:"kind" json:"kind"` // feature | shell
}

// Doc is the manifest file shape.
type Doc struct {
	Version        int       `yaml:"version" json:"version"`
	LastGenerated  string    `yaml:"lastGenerated,omitempty" json:"lastGenerated,omitempty"`
	Routes         []Route   `yaml:"routes" json:"routes"`
}

// RegisterPayload POST body for register action (aligned with TS POST JSON).
type RegisterPayload struct {
	SpecName     string  `json:"specName"`
	SpecPath     string  `json:"specPath"`
	DisplayTitle string  `json:"displayTitle,omitempty"`
	YAMLContent  string  `json:"yamlContent"`
	EntryHTML    *string `json:"entryHtml,omitempty"`
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9_-]+`)

func SlugFromSpecName(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	s = slugInvalid.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "spec-route"
	}
	return s
}

func basenameNoYAML(path string) string {
	p := filepath.Base(strings.ReplaceAll(path, `\`, `/`))
	p = strings.TrimSuffix(strings.TrimSuffix(p, ".yaml"), ".yml")
	return p
}

func getPrototypeFileRel(yamlContent string) string {
	var root map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return ""
	}
	proto, _ := root["prototype"].(map[string]interface{})
	if proto == nil {
		return ""
	}
	f, _ := proto["file"].(string)
	f = strings.TrimSpace(f)
	if f == "" {
		return ""
	}
	return strings.ReplaceAll(f, `\`, `/`)
}

func IsSafeWorkspaceRel(rel string) bool {
	if strings.TrimSpace(rel) == "" {
		return false
	}
	norm := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(rel), `\`, `/`), "/")
	if norm == "" || strings.Contains(norm, "..") {
		return false
	}
	return true
}

func DeriveEntryHTML(specPath, yamlContent string, override *string) string {
	if override != nil {
		o := strings.TrimSpace(*override)
		if o != "" && IsSafeWorkspaceRel(o) {
			return strings.ReplaceAll(o, `\`, `/`)
		}
	}
	fromY := getPrototypeFileRel(yamlContent)
	if fromY != "" && IsSafeWorkspaceRel(fromY) {
		return fromY
	}
	stem := basenameNoYAML(specPath)
	return filepath.ToSlash(filepath.Join(".vibex", "prototypes", stem+".html"))
}

func BuildFeatureRoute(specName, specPath, displayTitle, yamlContent string, entryOverride *string) Route {
	id := SlugFromSpecName(specName)
	title := strings.TrimSpace(displayTitle)
	if title == "" {
		title = specName
	}
	entry := DeriveEntryHTML(specPath, yamlContent, entryOverride)
	return Route{
		ID:        id,
		Path:      fmt.Sprintf("/proto/%s", id),
		Title:     title,
		SpecRef:   specName,
		EntryHTML: entry,
		Kind:      "feature",
	}
}

func NormalizeManifest(raw interface{}) Doc {
	out := Doc{Version: 1, Routes: []Route{}}
	m, ok := raw.(map[string]interface{})
	if !ok || m == nil {
		return out
	}
	if v, ok := m["version"].(int); ok {
		out.Version = v
	}
	if v, ok := m["version"].(float64); ok {
		out.Version = int(v)
	}
	if s, ok := m["lastGenerated"].(string); ok {
		out.LastGenerated = s
	}
	rawRoutes, _ := m["routes"].([]interface{})
	for _, rr := range rawRoutes {
		rm, ok := rr.(map[string]interface{})
		if !ok {
			continue
		}
		id := strings.TrimSpace(asString(rm["id"]))
		p := strings.TrimSpace(asString(rm["path"]))
		title := strings.TrimSpace(asString(rm["title"]))
		specRef := strings.TrimSpace(asString(rm["specRef"]))
		entryHTML := strings.TrimSpace(asString(rm["entryHtml"]))
		kind := strings.TrimSpace(asString(rm["kind"]))
		if kind != "shell" {
			kind = "feature"
		}
		if id == "" || p == "" || title == "" || specRef == "" || entryHTML == "" {
			continue
		}
		if !IsSafeWorkspaceRel(entryHTML) {
			continue
		}
		out.Routes = append(out.Routes, Route{
			ID: id, Path: p, Title: title, SpecRef: specRef, EntryHTML: entryHTML, Kind: kind,
		})
	}
	return out
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func upsertRoute(doc *Doc, route Route) error {
	for i := range doc.Routes {
		if doc.Routes[i].ID == route.ID {
			if doc.Routes[i].SpecRef != route.SpecRef {
				return fmt.Errorf("路由 id「%s」已被 spec「%s」占用，请改名或手工编辑 .vibex/prototype-manifest.yaml", route.ID, doc.Routes[i].SpecRef)
			}
			doc.Routes[i] = route
			return nil
		}
	}
	for i := range doc.Routes {
		if doc.Routes[i].SpecRef == route.SpecRef {
			doc.Routes[i] = route
			return nil
		}
	}
	doc.Routes = append(doc.Routes, route)
	return nil
}

// Get reads manifest from workspace root; exists=false if file missing.
func Get(root string) (exists bool, doc Doc, manifestPath string, err error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || root == "" {
		return false, Doc{}, "", fmt.Errorf("invalid root")
	}
	if _, err := os.Stat(rootAbs); err != nil {
		return false, Doc{}, "", fmt.Errorf("workspace not found")
	}
	mp := filepath.Join(rootAbs, filepath.FromSlash(ManifestRel))
	manifestPath = filepath.ToSlash(ManifestRel)
	data, err := os.ReadFile(mp)
	if err != nil {
		if os.IsNotExist(err) {
			return false, Doc{Version: 1, Routes: []Route{}}, manifestPath, nil
		}
		return false, Doc{}, manifestPath, err
	}
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, Doc{}, manifestPath, err
	}
	return true, NormalizeManifest(raw), manifestPath, nil
}

// Register merges route into manifest and writes file.
func Register(root string, payload RegisterPayload) (route Route, doc Doc, manifestPath string, err error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || root == "" {
		return Route{}, Doc{}, "", fmt.Errorf("invalid root")
	}
	if strings.TrimSpace(payload.SpecName) == "" || strings.TrimSpace(payload.SpecPath) == "" {
		return Route{}, Doc{}, "", errors.New("specName and specPath required")
	}
	title := strings.TrimSpace(payload.DisplayTitle)
	route = BuildFeatureRoute(payload.SpecName, payload.SpecPath, title, payload.YAMLContent, payload.EntryHTML)
	if !IsSafeWorkspaceRel(route.EntryHTML) {
		return Route{}, Doc{}, "", fmt.Errorf("entryHtml 路径非法（需相对工作区根，且不含 ..）")
	}

	mp := filepath.Join(rootAbs, filepath.FromSlash(ManifestRel))
	manifestPath = filepath.ToSlash(ManifestRel)

	var base Doc
	if data, err := os.ReadFile(mp); err == nil {
		var raw interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return Route{}, Doc{}, manifestPath, fmt.Errorf("读取 manifest 失败：%w", err)
		}
		base = NormalizeManifest(raw)
	} else if os.IsNotExist(err) {
		base = Doc{Version: 1, Routes: []Route{}}
	} else {
		return Route{}, Doc{}, manifestPath, err
	}

	if err := upsertRoute(&base, route); err != nil {
		return Route{}, Doc{}, manifestPath, err
	}
	base.LastGenerated = time.Now().UTC().Format(time.RFC3339Nano)
	base.Version = 1
	if base.Version == 0 {
		base.Version = 1
	}

	out, err := yaml.Marshal(&base)
	if err != nil {
		return Route{}, Doc{}, manifestPath, err
	}
	dir := filepath.Dir(mp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Route{}, Doc{}, manifestPath, err
	}
	if err := os.WriteFile(mp, out, 0644); err != nil {
		return Route{}, Doc{}, manifestPath, err
	}
	return route, base, manifestPath, nil
}
