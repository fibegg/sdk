package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

type launchSourceKind string

const (
	launchSourceNone            launchSourceKind = ""
	launchSourceTemplate        launchSourceKind = "template"
	launchSourceTemplateVersion launchSourceKind = "template_version"
	launchSourcePlayspec        launchSourceKind = "playspec"
	launchSourceCompose         launchSourceKind = "compose"
	launchSourceRepo            launchSourceKind = "repo"
)

type launchSource struct {
	Kind  launchSourceKind
	Value string
}

type launchSourceFlagValues struct {
	Template        string
	TemplateVersion string
	Playspec        string
	Compose         string
	Repo            string
}

func detectLaunchSource(cmd *cobra.Command, c *fibe.Client, args []string, values launchSourceFlagValues) (launchSource, error) {
	var selected []launchSource
	add := func(kind launchSourceKind, value string, changed bool) {
		value = strings.TrimSpace(value)
		if changed || value != "" {
			selected = append(selected, launchSource{Kind: kind, Value: value})
		}
	}
	add(launchSourceTemplate, values.Template, cmd.Flags().Changed("template"))
	add(launchSourceTemplateVersion, values.TemplateVersion, cmd.Flags().Changed("template-version"))
	add(launchSourcePlayspec, values.Playspec, cmd.Flags().Changed("playspec"))
	add(launchSourceCompose, values.Compose, cmd.Flags().Changed("compose"))
	add(launchSourceRepo, values.Repo, cmd.Flags().Changed("repo"))
	if len(args) > 0 {
		if len(args) > 1 {
			return launchSource{}, fmt.Errorf("expected at most one launch source")
		}
		selected = append(selected, launchSource{Kind: launchSourceNone, Value: strings.TrimSpace(args[0])})
	}
	if len(selected) == 0 {
		return launchSource{}, fmt.Errorf("launch source is required: use --template, --template-version, --playspec, --compose, or --repo")
	}
	if len(selected) > 1 {
		return launchSource{}, fmt.Errorf("provide exactly one launch source")
	}
	source := selected[0]
	if source.Kind != launchSourceNone {
		if source.Value == "" {
			return launchSource{}, fmt.Errorf("--%s cannot be blank", strings.ReplaceAll(string(source.Kind), "_", "-"))
		}
		return source, nil
	}
	return resolveBareLaunchSource(c, source.Value)
}

func resolveBareLaunchSource(c *fibe.Client, raw string) (launchSource, error) {
	if raw == "" {
		return launchSource{}, fmt.Errorf("launch source cannot be blank")
	}
	if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return launchSource{}, fmt.Errorf("bare numeric launch source %q is ambiguous; use --template %s or --playspec %s", raw, raw, raw)
	}
	if looksLikeRepositorySource(raw) {
		return launchSource{Kind: launchSourceRepo, Value: raw}, nil
	}

	templateMatch, templateErr := launchTemplateNameExists(c, raw)
	playspecMatch, playspecErr := launchPlayspecNameExists(c, raw)
	if templateErr != nil && playspecErr != nil {
		return launchSource{}, fmt.Errorf("could not resolve launch source %q as template or playspec: template: %v; playspec: %v", raw, templateErr, playspecErr)
	}
	switch {
	case templateMatch && playspecMatch:
		return launchSource{}, fmt.Errorf("launch source %q matches both a template and a playspec; use --template %q or --playspec %q", raw, raw, raw)
	case templateMatch:
		return launchSource{Kind: launchSourceTemplate, Value: raw}, nil
	case playspecMatch:
		return launchSource{Kind: launchSourcePlayspec, Value: raw}, nil
	default:
		return launchSource{}, fmt.Errorf("launch source %q was not found as a template or playspec; use --repo for repositories", raw)
	}
}

func looksLikeRepositorySource(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasPrefix(raw, "http://") ||
		strings.HasPrefix(raw, "https://") ||
		strings.Contains(raw, "/") ||
		strings.Contains(raw, ".git")
}

func launchTemplateNameExists(c *fibe.Client, name string) (bool, error) {
	result, err := c.ImportTemplates.List(ctx(), &fibe.ImportTemplateListParams{Name: name, PerPage: 2})
	if err != nil {
		return false, err
	}
	for _, item := range result.Data {
		if strings.EqualFold(item.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func launchPlayspecNameExists(c *fibe.Client, name string) (bool, error) {
	result, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Name: name, PerPage: 2})
	if err != nil {
		return false, err
	}
	for _, item := range result.Data {
		if strings.EqualFold(item.Name, name) {
			return true, nil
		}
	}
	return false, nil
}
