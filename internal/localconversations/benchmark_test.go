package localconversations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkWalkFileCandidates1000(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 1000; i++ {
		path := filepath.Join(root, fmt.Sprintf("conversation-%04d.jsonl", i))
		if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		files, err := walkFileCandidates(context.Background(), root, []string{root}, func(path string) bool {
			return filepath.Ext(path) == ".jsonl"
		}, "")
		if err != nil {
			b.Fatal(err)
		}
		if len(files) != 1000 {
			b.Fatalf("files=%d", len(files))
		}
	}
}
