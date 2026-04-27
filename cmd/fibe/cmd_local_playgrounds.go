package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fibegg/sdk/internal/localplaygrounds"
	"github.com/spf13/cobra"
)

type localService struct {
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

type localPlayground struct {
	DirName  string
	Playspec string
	Services map[string]*localService
}

func localPlaygroundsBaseDir() string {
	if v := os.Getenv("PLAYROOMS_ROOT"); v != "" {
		return v
	}
	return "/opt/fibe/playgrounds"
}

func localPlaygroundsRootDomain() string {
	if v := os.Getenv("MARQUEE_ROOT_DOMAIN"); v != "" {
		return v
	}
	return "phoenix.test"
}

var (
	rePlayspec   = regexp.MustCompile(`fibe\.gg/playspec:\s*['"]?([^\s'"]+)['"]?`)
	reService    = regexp.MustCompile(`^  ([a-zA-Z0-9_-]+):`)
	reImage      = regexp.MustCompile(`^\s+image:\s+['"]?([^'"]+)['"]?`)
	reTraefik    = regexp.MustCompile(`traefik\.enable:\s+['"]?true['"]?`)
	reExpose     = regexp.MustCompile(`fibe\.gg/expose:`)
	reSubdomain  = regexp.MustCompile(`fibe\.gg/subdomain:\s+['"]?([^'"]+)['"]?`)
	reStartCmd   = regexp.MustCompile(`fibe\.gg/start_command:\s+(.+)$`)
	reVolMount   = regexp.MustCompile(`^\s*-\s*["']?(/opt/fibe[^:"']+)[:"']`)
	reServicesHd = regexp.MustCompile(`^services:`)
	reTopLevel   = regexp.MustCompile(`^[a-zA-Z]`)
	rePropsMount = regexp.MustCompile(`-\s*["']?(.*?/props/([^/]+)/([^/:]+))[:"']`)
)

func parseComposeServices(content string) map[string]*localService {
	lines := strings.Split(content, "\n")
	services := make(map[string]*localService)
	var current *localService
	inServices := false

	for _, line := range lines {
		if reServicesHd.MatchString(line) {
			inServices = true
			continue
		}
		if inServices && reTopLevel.MatchString(line) {
			inServices = false
		}
		if !inServices {
			continue
		}

		if m := reService.FindStringSubmatch(line); m != nil {
			current = &localService{Name: m[1]}
			services[m[1]] = current
			continue
		}
		if current == nil {
			continue
		}

		if m := reImage.FindStringSubmatch(line); m != nil {
			current.Image = m[1]
		}
		if reTraefik.MatchString(line) {
			current.Traefik = true
		}
		if reExpose.MatchString(line) {
			current.Expose = true
		}
		if m := reSubdomain.FindStringSubmatch(line); m != nil {
			current.Subdomain = m[1]
		}
		if m := reStartCmd.FindStringSubmatch(line); m != nil {
			cmd := strings.TrimSpace(m[1])
			cmd = strings.TrimLeft(cmd, `"'`)
			cmd = strings.TrimRight(cmd, `"'`)
			current.StartCmd = cmd
		}
		if m := reVolMount.FindStringSubmatch(line); m != nil {
			hostPath := m[1]
			propsIdx := strings.Index(hostPath, "/props/")
			if propsIdx != -1 {
				relative := hostPath[propsIdx+7:]
				parts := strings.SplitN(relative, "/", 3)
				if len(parts) >= 2 {
					current.HostMount = hostPath
					rawProp := parts[0]
					propParts := strings.Split(rawProp, "--")
					if len(propParts) >= 3 {
						current.Prop = strings.Join(propParts[1:len(propParts)-1], "--")
					} else {
						current.Prop = rawProp
					}
					current.Branch = parts[1]
				}
			}
		}
	}

	return services
}

func scanPlaygrounds(baseDir string) ([]localPlayground, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("directory '%s' does not exist.\nYou can set the PLAYROOMS_ROOT environment variable", baseDir)
	}

	var playgrounds []localPlayground
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
		if m := rePlayspec.FindStringSubmatch(content); m != nil {
			playspec = m[1]
		}

		pg := localPlayground{
			DirName:  entry.Name(),
			Playspec: playspec,
			Services: parseComposeServices(content),
		}
		playgrounds = append(playgrounds, pg)
	}
	return playgrounds, nil
}

func findPlayground(playgrounds []localPlayground, target string) *localPlayground {
	for i := range playgrounds {
		pg := &playgrounds[i]
		if pg.DirName == target || pg.Playspec == target || strings.HasPrefix(pg.Playspec, target) {
			return pg
		}
	}
	return nil
}

func localPlaygroundsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "local-playgrounds",
		Aliases: []string{"lp"},
		Short:   "Explore local playgrounds on this machine",
		Long: `A CLI utility to explore local Playgrounds.

Reads compose.yml files from the playgrounds directory to discover
services, URLs, mount points, and other configuration.

Environment Variables:
  PLAYROOMS_ROOT        Directory containing the playgrounds (default: /opt/fibe/playgrounds)
  MARQUEE_ROOT_DOMAIN   Base domain for URLs (default: phoenix.test)

Examples:
  fibe local-playgrounds list                    List all playgrounds
  fibe local-playgrounds info my-app             Show extended info
  fibe local-playgrounds urls my-app             Output service URLs
  fibe local-playgrounds link my-app             Map playground into /app/playground`,
	}

	cmd.AddCommand(
		lpListCmd(),
		lpInfoCmd(),
		lpUrlsCmd(),
		lpLinkCmd(),
	)

	return cmd
}

func lpListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all local playgrounds",
		Long:  `List all playgrounds found in the playgrounds directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := localPlaygroundsBaseDir()
			playgrounds, err := scanPlaygrounds(baseDir)
			if err != nil {
				return err
			}

			switch effectiveOutput() {
			case "json", "yaml":
				type item struct {
					Name     string `json:"name" yaml:"name"`
					Playspec string `json:"playspec" yaml:"playspec"`
				}
				var items []item
				for _, pg := range playgrounds {
					items = append(items, item{Name: pg.DirName, Playspec: pg.Playspec})
				}
				output(items)
			default:
				for _, pg := range playgrounds {
					fmt.Printf("%s|%s\n", pg.DirName, pg.Playspec)
				}
			}
			return nil
		},
	}
}

func lpInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <playground>",
		Short: "Show detailed information for a playground",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := localPlaygroundsBaseDir()
			rootDomain := localPlaygroundsRootDomain()

			playgrounds, err := scanPlaygrounds(baseDir)
			if err != nil {
				return err
			}

			pg := findPlayground(playgrounds, args[0])
			if pg == nil {
				return fmt.Errorf("no playground found matching '%s'", args[0])
			}

			switch effectiveOutput() {
			case "json", "yaml":
				output(pg)
			default:
				// Extract top-level branch/mount/prop from first match
				ymlPath := filepath.Join(baseDir, pg.DirName, "compose.yml")
				data, _ := os.ReadFile(ymlPath)
				content := string(data)

				branch := "unknown"
				mountPath := "unknown"
				propFormatted := "unknown"
				if m := rePropsMount.FindStringSubmatch(content); m != nil {
					mountPath = m[1]
					propNameRaw := m[2]
					branch = m[3]
					propParts := strings.Split(propNameRaw, "--")
					if len(propParts) >= 3 {
						propFormatted = fmt.Sprintf("%s (id=%s)", propParts[0], propParts[len(propParts)-1])
					} else {
						propFormatted = propNameRaw
					}
				}

				fmt.Println("==========================================================")
				fmt.Printf("Playground:  %s\n", pg.Playspec)
				fmt.Printf("Branch:      %s\n", branch)
				fmt.Printf("Prop:        %s\n", propFormatted)
				fmt.Printf("Mount:       %s\n", mountPath)
				fmt.Println()
				fmt.Println("Services:")

				for svcName, svc := range pg.Services {
					fmt.Printf("  - %s:\n", svcName)
					if svc.Image != "" {
						fmt.Printf("      Image:      %s\n", svc.Image)
					}
					if svc.Traefik && svc.Subdomain != "" {
						fmt.Printf("      URL:        https://%s.%s\n", svc.Subdomain, rootDomain)
					} else {
						fmt.Println("      Network:    Internal only")
					}
					if svc.StartCmd != "" {
						fmt.Printf("      Command:    %s\n", svc.StartCmd)
					}
					if svc.HostMount != "" {
						fmt.Printf("      Mount:      %s\n", svc.HostMount)
					}
					fmt.Println()
				}
				fmt.Println("==========================================================")
			}
			return nil
		},
	}
}

func lpUrlsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "urls <playground>",
		Short: "Output service URLs for a playground",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := localPlaygroundsBaseDir()
			rootDomain := localPlaygroundsRootDomain()

			playgrounds, err := scanPlaygrounds(baseDir)
			if err != nil {
				return err
			}

			pg := findPlayground(playgrounds, args[0])
			if pg == nil {
				return fmt.Errorf("no playground found matching '%s'", args[0])
			}

			seen := make(map[string]bool)
			switch effectiveOutput() {
			case "json", "yaml":
				type urlEntry struct {
					Service string `json:"service" yaml:"service"`
					URL     string `json:"url" yaml:"url"`
				}
				var entries []urlEntry
				for svcName, svc := range pg.Services {
					if svc.Traefik && svc.Subdomain != "" {
						fullURL := svc.Subdomain + "." + rootDomain
						if !seen[fullURL] {
							seen[fullURL] = true
							entries = append(entries, urlEntry{Service: svcName, URL: fullURL})
						}
					}
				}
				output(entries)
			default:
				for svcName, svc := range pg.Services {
					if svc.Traefik && svc.Subdomain != "" {
						fullURL := svc.Subdomain + "." + rootDomain
						if !seen[fullURL] {
							seen[fullURL] = true
							fmt.Printf("%s|%s\n", svcName, fullURL)
						}
					}
				}
			}
			return nil
		},
	}
}

func lpLinkCmd() *cobra.Command {
	var linkDir string

	cmd := &cobra.Command{
		Use:   "link <playground>",
		Short: "Link playground directories into a target directory",
		Long: `Create symlinks from playground mount points into a target directory.

By default links to /app/playground. Use --link-dir to specify a different target.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := localPlaygroundsBaseDir()

			playgrounds, err := scanPlaygrounds(baseDir)
			if err != nil {
				return err
			}

			pg := findPlayground(playgrounds, args[0])
			if pg == nil {
				return fmt.Errorf("no playground found matching '%s'", args[0])
			}

			targetDir := linkDir
			if targetDir == "" {
				targetDir = "/app/playground"
			}

			fmt.Printf("Linking %s...\n", pg.DirName)
			lp := &localplaygrounds.Playground{
				DirName:  pg.DirName,
				Playspec: pg.Playspec,
				Services: make(map[string]*localplaygrounds.Service, len(pg.Services)),
			}
			for name, svc := range pg.Services {
				lp.Services[name] = &localplaygrounds.Service{
					Name:      svc.Name,
					Image:     svc.Image,
					Traefik:   svc.Traefik,
					Expose:    svc.Expose,
					Subdomain: svc.Subdomain,
					StartCmd:  svc.StartCmd,
					HostMount: svc.HostMount,
					Prop:      svc.Prop,
					Branch:    svc.Branch,
				}
			}
			result, err := localplaygrounds.LinkPlayground(lp, targetDir)
			if err != nil {
				return err
			}
			for _, link := range result.Links {
				fmt.Printf("Created symlink: %s -> %s\n", link.Path, link.Target)
			}
			fmt.Printf("State saved in %s\n", result.StateFile)

			return nil
		},
	}

	cmd.Flags().StringVar(&linkDir, "link-dir", "", "Target directory for symlinks (default: /app/playground)")
	return cmd
}
