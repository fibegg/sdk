package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fibegg/sdk/internal/localplaygrounds"
	"github.com/spf13/cobra"
)

func localPlaygroundsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "playgrounds",
		Aliases: []string{"lp"},
		Short:   "Explore local playgrounds on this machine",
		Long: `A CLI utility to explore local Playgrounds.

Reads compose.yml files from the playgrounds directory to discover
services, URLs, mount points, and other configuration.

Environment Variables:
  MARQUEE_ROOT          Marquee root, or directory containing playgrounds (default: /opt/fibe/playgrounds)
  MARQUEE_ROOT_DOMAIN   Base domain for URLs (default: phoenix.test)
  MARQUEE_URL_SCHEME    URL scheme for exposed services (default: https)

Examples:
  fibe local playgrounds info --view names
  fibe local playgrounds info --view current
  fibe local playgrounds info --view repos
  fibe local playgrounds info --view urls --playground 1
  fibe local playgrounds info --view mounts --playground mcp-test-dev
  fibe local playgrounds info --view details --playground 1
  fibe local playgrounds link 1 --link-dir /app/playground`,
	}

	cmd.AddCommand(
		lpInfoCmd(),
		lpLinkCmd(),
	)

	return cmd
}

func lpInfoCmd() *cobra.Command {
	var view string
	var playground string
	var linkDir string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show local playground information",
		Long: `Show local playground information.

Views:
  names    List selector-visible mountable local playground names, playspecs, IDs, and paths.
  current  Show the currently linked playground JSON state.
  repos    List git repository roots for the currently linked playground.
  urls     List exposed service URLs for one playground.
  mounts   List source-code mount locations for one playground.
  details  Show full local metadata for one playground.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			view = strings.ToLower(strings.TrimSpace(view))
			if view == "" {
				return fmt.Errorf("required flag '--view' not set")
			}
			selector, err := localPlaygroundSelector(view, playground)
			if err != nil {
				return err
			}

			playgrounds, err := localplaygrounds.Scan(localplaygrounds.BaseDir())
			if err != nil {
				return err
			}

			switch view {
			case "names":
				entries := localplaygrounds.Names(playgrounds)
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(entries)
					return nil
				}
				for _, entry := range entries {
					fmt.Printf("%s|%s|%s\n", entry.Name, entry.Playspec, entry.ID)
				}
			case "current":
				state, err := localplaygrounds.LoadCurrentState(linkDir)
				if err != nil {
					return err
				}
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(state)
					return nil
				}
				outputLocalCurrentState(state)
			case "repos":
				state, err := localplaygrounds.LoadCurrentState(linkDir)
				if err != nil {
					return err
				}
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(state.Repos)
					return nil
				}
				for _, entry := range state.Repos {
					fmt.Printf("%s|%s|%s|%s|%s\n", entry.Service, entry.Prop, entry.Branch, entry.LinkPath, entry.RepoRoot)
				}
			case "urls":
				pg, err := localplaygrounds.Find(playgrounds, selector)
				if err != nil {
					return err
				}
				entries := localplaygrounds.URLs(pg, localplaygrounds.RootDomain())
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(entries)
					return nil
				}
				for _, entry := range entries {
					fmt.Printf("%s|%s\n", entry.Service, entry.URL)
				}
			case "mounts":
				pg, err := localplaygrounds.Find(playgrounds, selector)
				if err != nil {
					return err
				}
				entries := localplaygrounds.Mounts(pg)
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(entries)
					return nil
				}
				for _, entry := range entries {
					fmt.Printf("%s|%s|%s|%s\n", entry.Service, entry.Mount, entry.Prop, entry.Branch)
				}
			case "details":
				pg, err := localplaygrounds.Find(playgrounds, selector)
				if err != nil {
					return err
				}
				if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
					output(pg)
					return nil
				}
				outputLocalPlaygroundDetails(pg)
			default:
				return fmt.Errorf("unknown local playground view %q (valid: %s)", view, strings.Join(localplaygrounds.Views, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&view, "view", "", "Info view: names, current, repos, urls, mounts, or details")
	cmd.Flags().StringVar(&playground, "playground", "", "Local playground ID, name, compose project, playspec, or unique playspec prefix")
	cmd.Flags().StringVar(&linkDir, "link-dir", "", "Current-link directory for views current and repos (default: /app/playground)")
	return cmd
}

func localPlaygroundSelector(view, playground string) (string, error) {
	playground = strings.TrimSpace(playground)
	if view == "names" || view == "current" || view == "repos" {
		if playground != "" {
			return "", fmt.Errorf("view '%s' does not accept --playground", view)
		}
		return "", nil
	}
	if playground != "" {
		return playground, nil
	}
	return "", fmt.Errorf("view '%s' requires --playground", view)
}

func outputLocalPlaygroundDetails(pg *localplaygrounds.Playground) {
	branch := "unknown"
	mountPath := "unknown"
	prop := "unknown"
	if mounts := localplaygrounds.Mounts(pg); len(mounts) > 0 {
		branch = valueOrUnknown(mounts[0].Branch)
		mountPath = valueOrUnknown(mounts[0].Mount)
		prop = valueOrUnknown(mounts[0].Prop)
	}

	fmt.Println("==========================================================")
	fmt.Printf("Playground:  %s\n", pg.Playspec)
	if pg.ID != "" {
		fmt.Printf("ID:          %s\n", pg.ID)
	}
	fmt.Printf("Name:        %s\n", pg.DirName)
	fmt.Printf("Branch:      %s\n", branch)
	fmt.Printf("Prop:        %s\n", prop)
	fmt.Printf("Mount:       %s\n", mountPath)
	fmt.Println()
	fmt.Println("Services:")

	for _, svcName := range localServiceNames(pg.Services) {
		svc := pg.Services[svcName]
		fmt.Printf("  - %s:\n", svcName)
		if svc.Image != "" {
			fmt.Printf("      Image:      %s\n", svc.Image)
		}
		if svc.Traefik && svc.Subdomain != "" {
			fmt.Printf("      URL:        %s://%s.%s\n", localplaygrounds.URLScheme(), svc.Subdomain, localplaygrounds.RootDomain())
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

func outputLocalCurrentState(state *localplaygrounds.CurrentState) {
	fmt.Println("==========================================================")
	fmt.Printf("Playground:  %s\n", state.Playspec)
	if state.ID != "" {
		fmt.Printf("ID:          %s\n", state.ID)
	}
	fmt.Printf("Name:        %s\n", state.DirName)
	fmt.Printf("Path:        %s\n", state.Path)
	fmt.Printf("Link Dir:    %s\n", state.LinkDir)
	fmt.Println()
	fmt.Println("Repositories:")
	for _, repo := range state.Repos {
		fmt.Printf("  - %s: %s -> %s\n", repo.Service, repo.LinkPath, repo.RepoRoot)
	}
	fmt.Println("==========================================================")
}

func localServiceNames(services map[string]*localplaygrounds.Service) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func lpLinkCmd() *cobra.Command {
	var linkDir string

	cmd := &cobra.Command{
		Use:   "link <id-or-name>",
		Short: "Link playground directories into a target directory",
		Long: `Create symlinks from playground mount points into a target directory.

By default links to /app/playground. Use --link-dir to specify a different target.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playgrounds, err := localplaygrounds.Scan(localplaygrounds.BaseDir())
			if err != nil {
				return err
			}

			pg, err := localplaygrounds.Find(playgrounds, args[0])
			if err != nil {
				return err
			}

			targetDir := linkDir
			if targetDir == "" {
				targetDir = "/app/playground"
			}

			fmt.Printf("Linking %s...\n", pg.DirName)
			result, err := localplaygrounds.LinkPlayground(pg, targetDir)
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
