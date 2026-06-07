package diff

import "testing"

func TestIsFullModeSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "go source", path: "internal/viewer/store.go", expected: true},
		{name: "tsx source", path: "pages/src/App.tsx", expected: true},
		{name: "python source", path: "apps/worker/task.py", expected: true},
		{name: "shell source", path: "scripts/deploy.sh", expected: true},
		{name: "package json", path: "package.json", expected: false},
		{name: "lock file", path: "pnpm-lock.yaml", expected: false},
		{name: "workflow yaml", path: ".github/workflows/release.yml", expected: false},
		{name: "readme", path: "README.md", expected: false},
		{name: "gitignore", path: ".gitignore", expected: false},
		{name: "env file", path: ".env", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFullModeSourceFile(tt.path)
			if got != tt.expected {
				t.Fatalf("isFullModeSourceFile(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
