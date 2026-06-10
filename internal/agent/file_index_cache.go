package agent

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type lightFileIndexCache struct {
	Version int                       `json:"version"`
	Entries map[string]lightFileEntry `json:"entries"`
}

func loadLightFileIndexCache(repoDir string, hints []string) lightFileIndex {
	path := lightFileIndexCachePath(repoDir, hints)
	data, err := os.ReadFile(path)
	if err != nil {
		return make(lightFileIndex)
	}
	var cache lightFileIndexCache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Version != 1 || cache.Entries == nil {
		return make(lightFileIndex)
	}
	return lightFileIndex(cache.Entries)
}

func saveLightFileIndexCache(repoDir string, hints []string, entries lightFileIndex) {
	path := lightFileIndexCachePath(repoDir, hints)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(lightFileIndexCache{Version: 1, Entries: map[string]lightFileEntry(entries)})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func lightFileIndexCachePath(repoDir string, hints []string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(repoDir, ".opencodereview", "cache", "light-index.json")
	}
	h := sha1.Sum([]byte(repoDir + "\x00" + strings.Join(hints, "\x00")))
	return filepath.Join(home, ".opencodereview", "cache", "light-index", hex.EncodeToString(h[:])+".json")
}
