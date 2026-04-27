package localplaygrounds

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

type Service struct {
	Name      string
	Image     string
	Traefik   bool
	Expose    bool
	Subdomain string
	StartCmd  string
	HostMount string
	Prop      string
	Branch    string
}

type Playground struct {
	DirName  string
	Playspec string
	Services map[string]*Service
}

func BaseDir() string {
	if v := os.Getenv("PLAYROOMS_ROOT"); v != "" {
		return v
	}
	return "/opt/fibe/playgrounds"
}

var (
	rePlayspecLocal   = regexp.MustCompile(`fibe\.gg/playspec:\s*['"]?([^\s'"]+)['"]?`)
	reServiceLocal    = regexp.MustCompile(`^  ([a-zA-Z0-9_-]+):`)
	reImageLocal      = regexp.MustCompile(`^\s+image:\s+['"]?([^'"]+)['"]?`)
	reTraefikLocal    = regexp.MustCompile(`traefik\.enable:\s+['"]?true['"]?`)
	reExposeLocal     = regexp.MustCompile(`fibe\.gg/expose:`)
	reSubdomainLocal  = regexp.MustCompile(`fibe\.gg/subdomain:\s+['"]?([^'"]+)['"]?`)
	reStartCmdLocal   = regexp.MustCompile(`fibe\.gg/start_command:\s+(.+)$`)
	reVolMountLocal   = regexp.MustCompile(`^\s*-\s*["']?(/opt/fibe[^:"']+)[:"']`)
	reServicesHdLocal = regexp.MustCompile(`^services:`)
	reTopLevelLocal   = regexp.MustCompile(`^[a-zA-Z]`)
)

func Scan(baseDir string) ([]Playground, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("directory '%s' does not exist.\nYou can set the PLAYROOMS_ROOT environment variable", baseDir)
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
		content := string(data)

		playspec := entry.Name()
		if m := rePlayspecLocal.FindStringSubmatch(content); m != nil {
			playspec = m[1]
		}

		playgrounds = append(playgrounds, Playground{
			DirName:  entry.Name(),
			Playspec: playspec,
			Services: parseServices(content),
		})
	}
	return playgrounds, nil
}

func Find(playgrounds []Playground, target string) *Playground {
	for i := range playgrounds {
		pg := &playgrounds[i]
		if pg.DirName == target || pg.Playspec == target || strings.HasPrefix(pg.Playspec, target) {
			return pg
		}
	}
	return nil
}

func Link(target, linkDir string) (*fibe.GreenfieldLinkResult, error) {
	if linkDir == "" {
		linkDir = "/app/playground"
	}
	playgrounds, err := Scan(BaseDir())
	if err != nil {
		return nil, err
	}
	pg := Find(playgrounds, target)
	if pg == nil {
		return nil, fmt.Errorf("no playground found matching '%s'", target)
	}
	return LinkPlayground(pg, linkDir)
}

func LinkPlayground(pg *Playground, linkDir string) (*fibe.GreenfieldLinkResult, error) {
	if linkDir == "" {
		linkDir = "/app/playground"
	}
	if info, err := os.Lstat(linkDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(linkDir); err != nil {
				return nil, fmt.Errorf("failed to remove existing symlink %s: %w", linkDir, err)
			}
		} else if err := os.RemoveAll(linkDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing target directory %s: %w", linkDir, err)
		}
	}
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
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
		StateFile:  filepath.Join(linkDir, ".current_playground"),
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

	if err := os.WriteFile(result.StateFile, []byte(pg.DirName), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write state file: %w", err)
	}
	return result, nil
}

func parseServices(content string) map[string]*Service {
	lines := strings.Split(content, "\n")
	services := make(map[string]*Service)
	var current *Service
	inServices := false

	for _, line := range lines {
		if reServicesHdLocal.MatchString(line) {
			inServices = true
			continue
		}
		if inServices && reTopLevelLocal.MatchString(line) {
			inServices = false
		}
		if !inServices {
			continue
		}

		if m := reServiceLocal.FindStringSubmatch(line); m != nil {
			current = &Service{Name: m[1]}
			services[m[1]] = current
			continue
		}
		if current == nil {
			continue
		}

		if m := reImageLocal.FindStringSubmatch(line); m != nil {
			current.Image = m[1]
		}
		if reTraefikLocal.MatchString(line) {
			current.Traefik = true
		}
		if reExposeLocal.MatchString(line) {
			current.Expose = true
		}
		if m := reSubdomainLocal.FindStringSubmatch(line); m != nil {
			current.Subdomain = m[1]
		}
		if m := reStartCmdLocal.FindStringSubmatch(line); m != nil {
			cmd := strings.TrimSpace(m[1])
			cmd = strings.TrimLeft(cmd, `"'`)
			cmd = strings.TrimRight(cmd, `"'`)
			current.StartCmd = cmd
		}
		if m := reVolMountLocal.FindStringSubmatch(line); m != nil {
			setMount(current, m[1])
		}
	}

	return services
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
