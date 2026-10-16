package localconversations

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

const (
	maxConversationScanFiles = 100_000
	maxConversationScanBytes = int64(4 << 30)
)

type fileCandidate struct {
	Path    string
	ModTime time.Time
}

func readJSONL(path string, handle func(map[string]any) error) error {
	// #nosec G304 -- path is selected by a strategy and accepted only after regular-file/symlink validation.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return fmt.Errorf("%s:%d: decode jsonl: %w", path, lineNo, err)
		}
		if err := handle(item); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: scan jsonl: %w", path, err)
	}
	return nil
}

func discoverFileCandidates(ctx context.Context, homeDir string, roots []string, accept func(string) bool, query string) ([]fileCandidate, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query != "" {
		return discoverQueryFileCandidates(ctx, homeDir, roots, accept, query)
	}
	return walkFileCandidates(ctx, homeDir, roots, accept, "")
}

func discoverQueryFileCandidates(ctx context.Context, homeDir string, roots []string, accept func(string) bool, query string) ([]fileCandidate, error) {
	allCandidates, err := walkFileCandidates(ctx, homeDir, roots, accept, "")
	if err != nil {
		return nil, err
	}
	byPath := map[string]fileCandidate{}
	for _, candidate := range allCandidates {
		if candidatePathMatchesQuery(candidate.Path, query) {
			byPath[candidate.Path] = candidate
			continue
		}
		matches, err := fileContainsQuery(ctx, candidate.Path, query)
		if err != nil {
			if shouldSkipPathError(err) {
				continue
			}
			return nil, err
		}
		if matches {
			byPath[candidate.Path] = candidate
		}
	}
	return sortedCandidateMap(byPath), nil
}

func walkFileCandidates(ctx context.Context, homeDir string, roots []string, accept func(string) bool, pathQuery string) ([]fileCandidate, error) {
	seen := make(map[string]bool)
	var files []fileCandidate
	var scannedFiles int
	var scannedBytes int64
	observe := func(path string, info fs.FileInfo) error {
		if !info.Mode().IsRegular() {
			return nil
		}
		scannedFiles++
		if info.Size() > 0 && scannedBytes > maxConversationScanBytes-info.Size() {
			return fmt.Errorf("local conversation scan exceeded the %d GiB aggregate budget at %s; narrow --path or provider selection", maxConversationScanBytes>>30, path)
		}
		scannedBytes += info.Size()
		if scannedFiles > maxConversationScanFiles {
			return fmt.Errorf("local conversation scan exceeded the %d-file budget at %s; narrow --path or provider selection", maxConversationScanFiles, path)
		}
		return nil
	}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = expandPath(root, homeDir)
		info, err := os.Lstat(root)
		if err != nil {
			if shouldSkipPathError(err) {
				continue
			}
			return nil, err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !info.IsDir() {
			if err := observe(root, info); err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			path, err := filepath.Abs(root)
			if err != nil {
				return nil, err
			}
			if accept(path) && candidatePathMatchesQuery(path, pathQuery) && !seen[path] {
				seen[path] = true
				files = append(files, fileCandidate{Path: path, ModTime: info.ModTime()})
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if shouldSkipPathError(walkErr) && d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if err := observe(path, info); err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if accept(abs) && candidatePathMatchesQuery(abs, pathQuery) && !seen[abs] {
				seen[abs] = true
				files = append(files, fileCandidate{Path: abs, ModTime: info.ModTime()})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sortFileCandidates(files)
	return files, nil
}

func candidatePathMatchesQuery(path, query string) bool {
	return query == "" || strings.Contains(strings.ToLower(path), query)
}

func fileContainsQuery(ctx context.Context, path, query string) (bool, error) {
	// #nosec G304 -- path is a budgeted, validated conversation file selected by the local scanner.
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if strings.Contains(strings.ToLower(scanner.Text()), query) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func sortedCandidateMap(values map[string]fileCandidate) []fileCandidate {
	out := make([]fileCandidate, 0, len(values))
	for _, candidate := range values {
		out = append(out, candidate)
	}
	sortFileCandidates(out)
	return out
}

func shouldSkipPathError(err error) bool {
	return os.IsNotExist(err) || os.IsPermission(err) || errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission)
}

func expandPath(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func sortFileCandidates(values []fileCandidate) {
	if len(values) < 2 {
		return
	}
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && fileCandidateLess(values[j], values[j-1]); j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func fileCandidateLess(left, right fileCandidate) bool {
	if !left.ModTime.Equal(right.ModTime) {
		return left.ModTime.After(right.ModTime)
	}
	return left.Path < right.Path
}

func prioritizeCandidates(candidates []fileCandidate, query string) {
	if len(candidates) < 2 {
		return
	}
	query = strings.ToLower(strings.TrimSpace(query))
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidatePriorityLess(candidates[j], candidates[j-1], query); j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
}

func candidatePriorityLess(left, right fileCandidate, query string) bool {
	leftContains := query != "" && strings.Contains(strings.ToLower(left.Path), query)
	rightContains := query != "" && strings.Contains(strings.ToLower(right.Path), query)
	if leftContains != rightContains {
		return leftContains
	}
	return fileCandidateLess(left, right)
}

func parseTimeValue(v any) *time.Time {
	switch x := v.(type) {
	case string:
		return parseTimeString(x)
	case float64:
		return unixMilliTime(int64(x))
	case int64:
		return unixMilliTime(x)
	case int:
		return unixMilliTime(int64(x))
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return unixMilliTime(n)
		}
	}
	return nil
}

func parseTimeString(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z0700"} {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return unixMilliTime(n)
	}
	return nil
}

func unixMilliTime(n int64) *time.Time {
	if n <= 0 {
		return nil
	}
	var t time.Time
	if n > 1_000_000_000_000 {
		t = time.UnixMilli(n).UTC()
	} else {
		t = time.Unix(n, 0).UTC()
	}
	return &t
}

func updateLastTime(current **time.Time, next *time.Time) {
	if next == nil {
		return
	}
	if *current == nil || next.After(**current) {
		t := *next
		*current = &t
	}
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolValue(v any) (bool, bool) {
	if b, ok := v.(bool); ok {
		return b, true
	}
	return false, false
}

func mapValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func int64Value(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	}
	return 0
}

func ensureMetadata(c *Conversation) map[string]any {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	return c.Metadata
}

func setMetadataString(c *Conversation, key string, value any) {
	s := stringValue(value)
	if s == "" {
		return
	}
	if _, exists := ensureMetadata(c)[key]; !exists {
		c.Metadata[key] = s
	}
}

func setMetadataBool(c *Conversation, key string, value any) {
	if b, ok := boolValue(value); ok {
		if _, exists := ensureMetadata(c)[key]; !exists {
			c.Metadata[key] = b
		}
	}
}

func messageMetadata(values map[string]any, keys ...string) map[string]any {
	metadata := make(map[string]any)
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		if s, ok := value.(string); ok && s == "" {
			continue
		}
		if slice, ok := value.([]any); ok && len(slice) == 0 {
			continue
		}
		if object, ok := value.(map[string]any); ok && len(object) == 0 {
			continue
		}
		metadata[key] = value
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func firstSentence(text string) string {
	text = normalizeWhitespace(text)
	if text == "" {
		return ""
	}

	runes := []rune(text)
	for i, r := range runes {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if i == len(runes)-1 || isSentenceSpace(runes[i+1]) {
			return strings.TrimSpace(string(runes[:i+1]))
		}
	}

	const maxRunes = 180
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func isSentenceSpace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func extractText(value any) string {
	switch x := value.(type) {
	case string:
		return x
	case []any:
		var parts []string
		for _, item := range x {
			part := extractTextBlock(item)
			if part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return extractTextBlock(x)
	default:
		return ""
	}
}

func extractTextBlock(value any) string {
	block := mapValue(value)
	if block == nil {
		return ""
	}

	blockType := stringValue(block["type"])
	if blockType != "" && blockType != "text" && blockType != "input_text" && blockType != "output_text" && blockType != "message" {
		return ""
	}

	for _, key := range []string{"text", "message", "content"} {
		if s := stringValue(block[key]); s != "" {
			return s
		}
	}
	return ""
}

func tokenUsageTotal(usage map[string]any) int64 {
	if usage == nil {
		return 0
	}
	if total := int64Value(usage["total_tokens"]); total > 0 {
		return total
	}
	var total int64
	for _, key := range []string{
		"input_tokens",
		"cache_creation_input_tokens",
		"cache_read_input_tokens",
		"cached_input_tokens",
		"output_tokens",
		"reasoning_output_tokens",
	} {
		total += int64Value(usage[key])
	}
	return total
}

func conversationIDMatchScore(id, query string) int {
	id = strings.ToLower(strings.TrimSpace(id))
	query = strings.ToLower(strings.TrimSpace(query))
	if id == "" || query == "" {
		return 0
	}
	if id == query {
		return 2
	}
	if strings.HasPrefix(id, query) {
		return 1
	}
	return 0
}

func uuidFromPath(path string) string {
	return uuidPattern.FindString(path)
}

func fileStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func hasJSONLExt(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".jsonl")
}

func hasJSONExt(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".json")
}
