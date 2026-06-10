package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-code-review/open-code-review/internal/model"
)

const lightIndexSampleBytes = 32 * 1024

type lightFileIndex map[string]lightFileEntry

type lightFileEntry struct {
	Path        string   `json:"path"`
	Summary     string   `json:"summary,omitempty"`
	Size        int64    `json:"size"`
	ModTimeUnix int64    `json:"modTimeUnix"`
	HintMatches []string `json:"hintMatches"`
}

func (a *Agent) buildLightFileIndex(diffs []model.Diff, hints []string) lightFileIndex {
	idx := make(lightFileIndex, len(diffs))

	cache := loadLightFileIndexCache(a.args.RepoDir, hints)
	changed := false
	for _, d := range diffs {
		path := effectivePath(d)
		fullPath := filepath.Join(a.args.RepoDir, path)
		stat, err := os.Stat(fullPath)
		if err != nil || stat.IsDir() {
			summary := summarizeFileSample(path, d.Diff)
			entry := lightFileEntry{Path: path, Summary: summary, HintMatches: matchHints(path, d.Diff+"\n"+summary, hints)}
			idx[path] = entry
			continue
		}

		if cached, ok := cache[path]; ok && cached.Size == stat.Size() && cached.ModTimeUnix == stat.ModTime().Unix() {
			idx[path] = cached
			continue
		}

		sample := readFilePrefix(fullPath, lightIndexSampleBytes)
		summary := summarizeFileSample(path, sample)
		entry := lightFileEntry{
			Path:        path,
			Summary:     summary,
			Size:        stat.Size(),
			ModTimeUnix: stat.ModTime().Unix(),
			HintMatches: matchHints(path, sample+"\n"+summary, hints),
		}
		idx[path] = entry
		cache[path] = entry
		changed = true
	}

	if changed {
		saveLightFileIndexCache(a.args.RepoDir, hints, cache)
	}
	return idx
}

func summarizeFileSample(path, sample string) string {
	if sample == "" {
		return filepath.Base(path)
	}
	var summary []string
	for _, line := range strings.Split(sample, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "+"))
		lower := strings.ToLower(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(lower, "func ") || strings.HasPrefix(lower, "type ") || strings.HasPrefix(lower, "class ") || strings.HasPrefix(lower, "interface ") || strings.HasPrefix(lower, "export ") || strings.Contains(lower, "router") || strings.Contains(lower, "route") {
			summary = append(summary, line)
		}
		if len(summary) >= 20 {
			break
		}
	}
	if len(summary) == 0 {
		return filepath.Base(path)
	}
	return strings.Join(summary, "\n")
}

func readFilePrefix(path string, limit int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, limit)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return ""
	}
	buf = buf[:n]
	if bytes.Contains(buf, []byte{0}) {
		return ""
	}
	return string(buf)
}

func matchHints(path, content string, hints []string) []string {
	lowerPath := strings.ToLower(path)
	lowerContent := strings.ToLower(content)
	matches := make([]string, 0, len(hints))
	seen := make(map[string]struct{}, len(hints))
	for _, hint := range hints {
		hint = strings.ToLower(strings.TrimSpace(hint))
		if hint == "" {
			continue
		}
		if _, ok := seen[hint]; ok {
			continue
		}
		if strings.Contains(lowerPath, hint) || strings.Contains(lowerContent, hint) {
			matches = append(matches, hint)
			seen[hint] = struct{}{}
		}
	}
	return matches
}
