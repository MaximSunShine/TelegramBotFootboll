package main

import "testing"

func TestEscapePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"main.go", "main.go"},
		{"cmd/bot/main.go", "cmd/bot/main.go"},
		{"file with space.go", "file%20with%20space.go"},
		{"кириллица/файл.go", "%D0%BA%D0%B8%D1%80%D0%B8%D0%BB%D0%BB%D0%B8%D1%86%D0%B0/%D1%84%D0%B0%D0%B9%D0%BB.go"},
		{"path/with#hash.go", "path/with%23hash.go"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := escapePath(tt.in)
			if got != tt.want {
				t.Errorf("escapePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
