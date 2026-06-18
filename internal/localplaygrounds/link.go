package localplaygrounds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"gopkg.in/yaml.v3"
)

const defaultBaseDir = "/opt/fibe/playgrounds"
const defaultRootDomain = "phoenix.test"
const defaultLinkDir = "/app/playground"
const currentStateFilename = ".current_playground.json"

var Views = []string{"names", "current", "repos", "urls", "mounts", "details"}

type Service struct {
	Name      string `json:"name" yaml:"name"`
	Image     string `json:"image,omitempty" yaml:"image,omitempty"`
	Traefik   bool   `json:"traefik,omitempty" yaml:"traefik,omitempty"`
	Expose    bool   `json:"expose,omitempty" yaml:"expose,omitempty"`
	Subdomain string `json:"subdomain,omitempty" yaml:"subdomain,omitempty"`
	StartCmd  string `json:"start_cmd,omitempty" yaml:"start_cmd,omitempty"`
	HostMount string `json:"host_mount,omitempty" yaml:"host_mount,omitempty"`
	Prop      string `json:"prop,omitempty" yaml:"prop,omitempty"`
	Branch    string `json:"branch,omitempty" yaml:"branch,omitempty"`
	JobWatch  bool   `json:"job_watch,omitempty" yaml:"job_watch,omitempty"`
}

type Playground struct {
	ID       string              `json:"id,omitempty" yaml:"id,omitempty"`
	DirName  string              `json:"name" yaml:"name"`
	Path     string              `json:"path" yaml:"path"`
	Playspec string              `json:"playspec" yaml:"playspec"`
	JobMode  bool                `json:"job_mode,omitempty" yaml:"job_mode,omitempty"`
	Services map[string]*Service `json:"services" yaml:"services"`
}

type NameEntry struct {
	ID       string `json:"id,omitempty" yaml:"id,omitempty"`
	Name     string `json:"name" yaml:"name"`
	Playspec string `json:"playspec" yaml:"playspec"`
	Path     string `json:"path" yaml:"path"`
}

type URLEntry struct {
	Service string `json:"service" yaml:"service"`
	URL     string `json:"url" yaml:"url"`
}

type MountEntry struct {
	Service string `json:"service" yaml:"service"`
	Mount   string `json:"mount" yaml:"mount"`
	Prop    string `json:"prop,omitempty" yaml:"prop,omitempty"`
	Branch  string `json:"branch,omitempty" yaml:"branch,omitempty"`
}

type RepoEntry struct {
	ID       string `json:"id,omitempty" yaml:"id,omitempty"`
	Service  string `json:"service" yaml:"service"`
	Prop     string `json:"prop,omitempty" yaml:"prop,omitempty"`
	Branch   string `json:"branch,omitempty" yaml:"branch,omitempty"`
	LinkPath string `json:"link_path" yaml:"link_path"`
	Target   string `json:"target" yaml:"target"`
	RepoRoot string `json:"repo_root" yaml:"repo_root"`
}

type CurrentState struct {
	ID         string              `json:"id,omitempty" yaml:"id,omitempty"`
	Name       string              `json:"name" yaml:"name"`
	DirName    string              `json:"dir_name" yaml:"dir_name"`
	Path       string              `json:"path" yaml:"path"`
	Playspec   string              `json:"playspec" yaml:"playspec"`
	LinkDir    string              `json:"link_dir" yaml:"link_dir"`
	StateFile  string              `json:"state_file" yaml:"state_file"`
	JobMode    bool                `json:"job_mode,omitempty" yaml:"job_mode,omitempty"`
	Services   map[string]*Service `json:"services" yaml:"services"`
	URLs       []URLEntry          `json:"urls" yaml:"urls"`
	Mounts     []MountEntry        `json:"mounts" yaml:"mounts"`
	Repos      []RepoEntry         `json:"repos" yaml:"repos"`
	RootDomain string              `json:"root_domain" yaml:"root_domain"`
}

type BaseDirMissingError struct {
	Path string
	Err  error
}

func (e *BaseDirMissingError) Error() string {
	return fmt.Sprintf("directory %q does not exist; set MARQUEE_ROOT to the Marquee root or playgrounds directory", e.Path)
}

func (e *BaseDirMissingError) Unwrap() error {
	return e.Err
}

func (e *BaseDirMissingError) ErrorCode() string {
	return "LOCAL_PLAYGROUNDS_DIR_MISSING"
}

func (e *BaseDirMissingError) ErrorStatus() int {
	return 404
}

func (e *BaseDirMissingError) ErrorDetails() map[string]any {
	return map[string]any{"path": e.Path}
}

func BaseDir() string {
	if v := strings.TrimSpace(os.Getenv("MARQUEE_ROOT")); v != "" {
		return resolveMarqueeRoot(v)
	}
	return defaultBaseDir
}

func RootDomain() string {
	if v := os.Getenv("MARQUEE_ROOT_DOMAIN"); v != "" {
		return v
	}
	return defaultRootDomain
}

func Scan(baseDir string) ([]Playground, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &BaseDirMissingError{Path: baseDir, Err: err}
		}
		return nil, err
	}

	var playgrounds []Playground
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ymlPath := filepath.Join(baseDir, entry.Name(), "compose.yml")
		data, err := os.ReadFile(ymlPath)
		if err != nil {
			continue
		}
		pg := parseCompose(entry.Name(), filepath.Join(baseDir, entry.Name()), data)
		playgrounds = append(playgrounds, pg)
	}
	sort.Slice(playgrounds, func(i, j int) bool {
		return playgrounds[i].DirName < playgrounds[j].DirName
	})
	return playgrounds, nil
}

func resolveMarqueeRoot(root string) string {
	root = filepath.Clean(root)
	if filepath.Base(root) == "playgrounds" || hasComposeProjectDirs(root) {
		return root
	}
	candidate := filepath.Join(root, "playgrounds")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return root
}

func hasComposeProjectDirs(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "compose.yml")); err == nil {
			return true
		}
	}
	return false
}

func Find(playgrounds []Playground, target string) (*Playground, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("required playground selector not set")
	}

	if isNumeric(target) {
		if pg := findByID(playgrounds, target); pg != nil {
			return pg, nil
		}
		if matches := exactNameMatches(playgrounds, target); len(matches) == 1 {
			return matches[0], nil
		} else if len(matches) > 1 {
			return nil, ambiguousError(target, matches)
		}
		return nil, fmt.Errorf("no playground found matching '%s'", target)
	}

	if matches := exactNameMatches(playgrounds, target); len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, ambiguousError(target, matches)
	}

	if matches := exactPlayspecMatches(playgrounds, target); len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, ambiguousError(target, matches)
	}

	if matches := playspecPrefixMatches(playgrounds, target); len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, ambiguousError(target, matches)
	}

	return nil, fmt.Errorf("no playground found matching '%s'", target)
}

func Names(playgrounds []Playground) []NameEntry {
	items := make([]NameEntry, 0, len(playgrounds))
	for _, pg := range playgrounds {
		if pg.JobMode || !hasMountableService(pg) {
			continue
		}
		items = append(items, NameEntry{
			ID:       pg.ID,
			Name:     pg.DirName,
			Playspec: pg.Playspec,
			Path:     pg.Path,
		})
	}
	return items
}

func hasMountableService(pg Playground) bool {
	for _, svc := range pg.Services {
		if svc.HostMount != "" {
			return true
		}
	}
	return false
}

func URLs(pg *Playground, rootDomain string) []URLEntry {
	if rootDomain == "" {
		rootDomain = defaultRootDomain
	}
	scheme := URLScheme()
	seen := make(map[string]bool)
	var entries []URLEntry
	for _, name := range serviceNames(pg.Services) {
		svc := pg.Services[name]
		if svc.Traefik && svc.Subdomain != "" {
			fullURL := scheme + "://" + svc.Subdomain + "." + rootDomain
			if !seen[fullURL] {
				seen[fullURL] = true
				entries = append(entries, URLEntry{Service: name, URL: fullURL})
			}
		}
	}
	return entries
}

func Mounts(pg *Playground) []MountEntry {
	var entries []MountEntry
	for _, name := range serviceNames(pg.Services) {
		svc := pg.Services[name]
		if svc.HostMount == "" {
			continue
		}
		entries = append(entries, MountEntry{
			Service: name,
			Mount:   svc.HostMount,
			Prop:    svc.Prop,
			Branch:  svc.Branch,
		})
	}
	return entries
}

func URLScheme() string {
	if v := strings.TrimSpace(os.Getenv("MARQUEE_URL_SCHEME")); v != "" {
		return strings.TrimSuffix(strings.ToLower(v), "://")
	}
	return "https"
}

func CurrentStatePath(linkDir string) string {
	if linkDir == "" {
		linkDir = defaultLinkDir
	}
	return filepath.Join(linkDir, currentStateFilename)
}

func LoadCurrentState(linkDir string) (*CurrentState, error) {
	path := CurrentStatePath(linkDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state CurrentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func View(playgrounds []Playground, view, selector, rootDomain, linkDir string) (any, error) {
	switch view {
	case "names":
		if strings.TrimSpace(selector) != "" {
			return nil, fmt.Errorf("view 'names' does not accept a playground selector")
		}
		return Names(playgrounds), nil
	case "current":
		if strings.TrimSpace(selector) != "" {
			return nil, fmt.Errorf("view 'current' does not accept a playground selector")
		}
		return LoadCurrentState(linkDir)
	case "repos":
		if strings.TrimSpace(selector) != "" {
			return nil, fmt.Errorf("view 'repos' does not accept a playground selector")
		}
		state, err := LoadCurrentState(linkDir)
		if err != nil {
			return nil, err
		}
		return state.Repos, nil
	case "urls":
		pg, err := Find(playgrounds, selector)
		if err != nil {
			return nil, err
		}
		return URLs(pg, rootDomain), nil
	case "mounts":
		pg, err := Find(playgrounds, selector)
		if err != nil {
			return nil, err
		}
		return Mounts(pg), nil
	case "details":
		pg, err := Find(playgrounds, selector)
		if err != nil {
			return nil, err
		}
		return pg, nil
	default:
		return nil, fmt.Errorf("unknown local playground view %q (valid: %s)", view, strings.Join(Views, ", "))
	}
}

func Link(target, linkDir string) (*fibe.GreenfieldLinkResult, error) {
	if linkDir == "" {
		linkDir = defaultLinkDir
	}
	playgrounds, err := Scan(BaseDir())
	if err != nil {
		return nil, err
	}
	pg, err := Find(playgrounds, target)
	if err != nil {
		return nil, err
	}
	return LinkPlayground(pg, linkDir)
}

func LinkPlayground(pg *Playground, linkDir string) (*fibe.GreenfieldLinkResult, error) {
	if linkDir == "" {
		linkDir = defaultLinkDir
	}
	if pg.JobMode {
		return nil, fmt.Errorf("cannot link job-mode playground %s", pg.DirName)
	}
	if err := prepareLinkDir(linkDir); err != nil {
		return nil, err
	}

	var mountable []*Service
	for _, svc := range pg.Services {
		if svc.HostMount != "" {
			mountable = append(mountable, svc)
		}
	}

	branches := make(map[string]bool)
	for _, svc := range mountable {
		branches[svc.Branch] = true
	}
	appendBranch := len(branches) > 1

	result := &fibe.GreenfieldLinkResult{
		LinkDir:    linkDir,
		Playground: pg.DirName,
		StateFile:  CurrentStatePath(linkDir),
	}
	created := make(map[string]bool)
	for _, svc := range mountable {
		if created[svc.HostMount] {
			continue
		}
		created[svc.HostMount] = true

		symlinkName := svc.Prop
		if symlinkName == "" {
			symlinkName = "default"
		}
		if appendBranch && svc.Branch != "" {
			symlinkName = symlinkName + "-" + svc.Branch
		}
		symlinkPath := filepath.Join(linkDir, symlinkName)

		if err := os.RemoveAll(symlinkPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing symlink %s: %w", symlinkPath, err)
		}
		if err := os.Symlink(svc.HostMount, symlinkPath); err != nil {
			return nil, fmt.Errorf("failed to create symlink %s -> %s: %w", symlinkPath, svc.HostMount, err)
		}
		result.Links = append(result.Links, fibe.GreenfieldLinkedPath{
			Name:    symlinkName,
			Path:    symlinkPath,
			Target:  svc.HostMount,
			Service: svc.Name,
			Prop:    svc.Prop,
			Branch:  svc.Branch,
		})
	}

	state := currentStateFromLinks(pg, linkDir, result.StateFile, result.Links)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize state file: %w", err)
	}
	if err := os.WriteFile(result.StateFile, append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write state file: %w", err)
	}
	return result, nil
}

func currentStateFromLinks(pg *Playground, linkDir, stateFile string, links []fibe.GreenfieldLinkedPath) CurrentState {
	repos := make([]RepoEntry, 0, len(links))
	for _, link := range links {
		repoRoot := findGitRoot(link.Target)
		if repoRoot == "" {
			repoRoot = link.Target
		}
		repos = append(repos, RepoEntry{
			ID:       link.Name,
			Service:  link.Service,
			Prop:     link.Prop,
			Branch:   link.Branch,
			LinkPath: link.Path,
			Target:   link.Target,
			RepoRoot: repoRoot,
		})
	}
	return CurrentState{
		ID:         pg.ID,
		Name:       pg.DirName,
		DirName:    pg.DirName,
		Path:       pg.Path,
		Playspec:   pg.Playspec,
		LinkDir:    linkDir,
		StateFile:  stateFile,
		JobMode:    pg.JobMode,
		Services:   pg.Services,
		URLs:       URLs(pg, RootDomain()),
		Mounts:     Mounts(pg),
		Repos:      repos,
		RootDomain: RootDomain(),
	}
}

func findGitRoot(start string) string {
	current := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func prepareLinkDir(linkDir string) error {
	info, err := os.Lstat(linkDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(linkDir, 0o755); err != nil {
				return fmt.Errorf("failed to create target directory %s: %w", linkDir, err)
			}
			return nil
		}
		return fmt.Errorf("failed to inspect target directory %s: %w", linkDir, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("target path %s must be a directory, got symlink", linkDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %s must be a directory", linkDir)
	}

	entries, err := os.ReadDir(linkDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %s: %w", linkDir, err)
	}
	for _, entry := range entries {
		path := filepath.Join(linkDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove existing target entry %s: %w", path, err)
		}
	}
	return nil
}

func parseCompose(dirName, dirPath string, data []byte) Playground {
	pg := Playground{
		DirName:  dirName,
		Path:     dirPath,
		Playspec: dirName,
		Services: map[string]*Service{},
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return pg
	}

	pg.JobMode = composeJobMode(doc)
	services := asMap(doc["services"])
	for _, svcName := range sortedMapKeys(services) {
		svcMap := asMap(services[svcName])
		labels := normalizeLabels(svcMap["labels"])
		service := &Service{
			Name:      svcName,
			Image:     scalarString(svcMap["image"]),
			Traefik:   truthy(labels["traefik.enable"]),
			Expose:    labelExists(labels, "fibe.gg/port"),
			Subdomain: labels["fibe.gg/subdomain"],
			StartCmd:  trimCommand(labels["fibe.gg/start_command"]),
			JobWatch:  truthy(labels["fibe.gg/job_watch"]),
		}
		if service.JobWatch {
			pg.JobMode = true
		}
		if pg.Playspec == dirName {
			if playspec := labels["fibe.gg/playspec"]; playspec != "" {
				pg.Playspec = playspec
			}
		}
		if pg.ID == "" {
			if playgroundName := labels["fibe.gg/playground"]; playgroundName != "" {
				pg.ID = trailingID(playgroundName)
			}
		}
		for _, volume := range volumeSources(svcMap["volumes"]) {
			setMount(service, volume)
			if service.HostMount != "" {
				break
			}
		}
		pg.Services[svcName] = service
	}
	if id := trailingID(dirName); id != "" {
		pg.ID = id
	}
	return pg
}

func composeJobMode(doc map[string]any) bool {
	namespace := asMap(doc["x-fibe.gg"])
	metadata := asMap(namespace["metadata"])
	return truthy(scalarString(metadata["job_mode"])) || truthy(scalarString(namespace["job_mode"]))
}

func asMap(value any) map[string]any {
	out := map[string]any{}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		for key, val := range typed {
			out[fmt.Sprint(key)] = val
		}
	}
	return out
}

func normalizeLabels(value any) map[string]string {
	labels := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for key, val := range typed {
			labels[key] = scalarString(val)
		}
	case map[any]any:
		for key, val := range typed {
			labels[fmt.Sprint(key)] = scalarString(val)
		}
	case []any:
		for _, item := range typed {
			label := scalarString(item)
			if label == "" {
				continue
			}
			key, val, ok := strings.Cut(label, "=")
			if !ok {
				key, val, ok = strings.Cut(label, ":")
			}
			if ok {
				labels[strings.TrimSpace(key)] = strings.TrimSpace(val)
			}
		}
	case []string:
		for _, label := range typed {
			key, val, ok := strings.Cut(label, "=")
			if !ok {
				key, val, ok = strings.Cut(label, ":")
			}
			if ok {
				labels[strings.TrimSpace(key)] = strings.TrimSpace(val)
			}
		}
	}
	return labels
}

func volumeSources(value any) []string {
	var out []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if source := volumeSource(item); source != "" {
				out = append(out, source)
			}
		}
	case []string:
		for _, item := range typed {
			if source := volumeSource(item); source != "" {
				out = append(out, source)
			}
		}
	}
	return out
}

func volumeSource(value any) string {
	switch typed := value.(type) {
	case string:
		source, _, _ := strings.Cut(strings.TrimSpace(typed), ":")
		return strings.Trim(source, `"'`)
	case map[string]any:
		return scalarString(typed["source"])
	case map[any]any:
		return scalarString(typed["source"])
	}
	return ""
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func trimCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.TrimLeft(cmd, `"'`)
	cmd = strings.TrimRight(cmd, `"'`)
	return cmd
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

func labelExists(labels map[string]string, key string) bool {
	value, ok := labels[key]
	return ok && strings.TrimSpace(value) != "false"
}

func setMount(service *Service, hostPath string) {
	propsIdx := strings.Index(hostPath, "/props/")
	if propsIdx == -1 {
		return
	}
	relative := hostPath[propsIdx+7:]
	parts := strings.SplitN(relative, "/", 3)
	if len(parts) < 2 {
		return
	}
	service.HostMount = hostPath
	rawProp := parts[0]
	propParts := strings.Split(rawProp, "--")
	if len(propParts) >= 3 {
		service.Prop = strings.Join(propParts[1:len(propParts)-1], "--")
	} else {
		service.Prop = rawProp
	}
	service.Branch = parts[1]
}

func trailingID(name string) string {
	idx := strings.LastIndex(name, "--")
	if idx == -1 || idx+2 >= len(name) {
		return ""
	}
	id := name[idx+2:]
	if !isNumeric(id) {
		return ""
	}
	return id
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func findByID(playgrounds []Playground, target string) *Playground {
	for i := range playgrounds {
		if playgrounds[i].ID == target {
			return &playgrounds[i]
		}
	}
	return nil
}

func exactNameMatches(playgrounds []Playground, target string) []*Playground {
	var matches []*Playground
	for i := range playgrounds {
		if playgrounds[i].DirName == target {
			matches = append(matches, &playgrounds[i])
		}
	}
	return matches
}

func exactPlayspecMatches(playgrounds []Playground, target string) []*Playground {
	var matches []*Playground
	for i := range playgrounds {
		if playgrounds[i].Playspec == target {
			matches = append(matches, &playgrounds[i])
		}
	}
	return matches
}

func playspecPrefixMatches(playgrounds []Playground, target string) []*Playground {
	var matches []*Playground
	for i := range playgrounds {
		if strings.HasPrefix(playgrounds[i].Playspec, target) {
			matches = append(matches, &playgrounds[i])
		}
	}
	return matches
}

func ambiguousError(target string, matches []*Playground) error {
	candidates := make([]string, 0, len(matches))
	for _, pg := range matches {
		label := pg.DirName
		if pg.Playspec != "" {
			label += " (playspec: " + pg.Playspec
			if pg.ID != "" {
				label += ", id: " + pg.ID
			}
			label += ")"
		}
		candidates = append(candidates, label)
	}
	sort.Strings(candidates)
	return fmt.Errorf("multiple playgrounds found matching '%s': %s", target, strings.Join(candidates, ", "))
}

func serviceNames(services map[string]*Service) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
