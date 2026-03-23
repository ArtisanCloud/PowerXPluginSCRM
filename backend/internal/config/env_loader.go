package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func loadEnvFiles(extraDirs ...string) {
	candidates := envCandidatePaths(extraDirs...)
	if len(candidates) == 0 {
		return
	}

	for _, path := range candidates {
		loadEnvFile(path)
	}
}

func envCandidatePaths(extraDirs ...string) []string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = ""
	}

	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}

	if cwd != "" {
		dir := cwd
		for i := 0; i < 6; i++ {
			add(filepath.Join(dir, ".env"))
			add(filepath.Join(dir, ".env.local"))
			add(filepath.Join(dir, "skeleton", "backend", ".env"))
			add(filepath.Join(dir, "skeleton", "backend", ".env.local"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, dir := range extraDirs {
		add(filepath.Join(dir, ".env"))
		add(filepath.Join(dir, ".env.local"))
	}

	return out
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	loaded := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, val, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		_ = os.Setenv(key, val)
		loaded++
	}

	if loaded > 0 {
		logrus.WithFields(logrus.Fields{
			"env_file":       path,
			"loaded_entries": loaded,
		}).Info("ENV file loaded")
	}
}

func parseEnvLine(line string) (string, string, bool) {
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", false
	}

	val := strings.TrimSpace(line[idx+1:])
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	return key, val, true
}
