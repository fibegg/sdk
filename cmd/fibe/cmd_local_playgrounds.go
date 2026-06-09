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

Examples:
  fibe local playgrounds info --view names
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
	var playgroundID string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show local playground information",
		Long: `Show local playground information.

Views:
  names    List selector-visible local playground names, playspecs, IDs, and paths.
  urls     List exposed service URLs for one playground.
  mounts   List source-code mount locations for one playground.
  details  Show full local metadata for one playground.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			view = strings.ToLower(strings.TrimSpace(view))
			if view == "" {
				return fmt.Errorf("required flag '--view' not set")
			}
			selector, err := localPlaygroundSelector(view, playground, playgroundID)
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

	cmd.Flags().StringVar(&view, "view", "", "Info view: names, urls, mounts, or details")
	cmd.Flags().StringVar(&playground, "playground", "", "Local playground ID, name, compose project, playspec, or unique playspec prefix")
	cmd.Flags().StringVar(&playgroundID, "playground-id", "", "Local playground numeric ID")
	return cmd
}

func localPlaygroundSelector(view, playground, playgroundID string) (string, error) {
	playground = strings.TrimSpace(playground)
	playgroundID = strings.TrimSpace(playgroundID)
	if view == "names" {
		if playground != "" || playgroundID != "" {
			return "", fmt.Errorf("view 'names' does not accept --playground or --playground-id")
		}
		return "", nil
	}
	if playground != "" && playgroundID != "" {
		return "", fmt.Errorf("pass only one of --playground or --playground-id")
	}
	if playgroundID != "" {
		return playgroundID, nil
	}
	if playground != "" {
		return playground, nil
	}
	return "", fmt.Errorf("view '%s' requires --playground or --playground-id", view)
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
			fmt.Printf("      URL:        https://%s.%s\n", svc.Subdomain, localplaygrounds.RootDomain())
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
