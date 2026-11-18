package precheck

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DefaultAuthFiles lists the container auth files podman/docker use, in lookup order.
func DefaultAuthFiles() []string {
	var files []string
	if f := os.Getenv("REGISTRY_AUTH_FILE"); f != "" {
		files = append(files, f)
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		files = append(files, filepath.Join(d, "containers", "auth.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files,
			filepath.Join(home, ".config", "containers", "auth.json"),
			filepath.Join(home, ".docker", "config.json"))
	}
	return files
}

// FindRegistryAuth returns the auth file and the most specific auth key that
// covers repo (e.g. "quay.io/me/ibu-seed" matches "quay.io/me" or "quay.io").
// Keys without an inline "auth" value (credential helpers) are not counted.
func FindRegistryAuth(repo string, files []string) (file, key string, err error) {
	repo = strings.TrimPrefix(strings.TrimPrefix(repo, "docker://"), "https://")
	if i := strings.LastIndex(repo, ":"); i > strings.LastIndex(repo, "/") {
		repo = repo[:i] // drop :tag
	}
	for _, f := range files {
		b, rerr := os.ReadFile(f) // #nosec G304 -- user's own auth files
		if rerr != nil {
			continue
		}
		var cfg struct {
			Auths map[string]struct {
				Auth string `json:"auth"`
			} `json:"auths"`
		}
		if json.Unmarshal(b, &cfg) != nil {
			continue
		}
		best := ""
		for k, v := range cfg.Auths {
			if v.Auth == "" {
				continue
			}
			k2 := strings.TrimPrefix(strings.TrimPrefix(k, "https://"), "http://")
			k2 = strings.TrimSuffix(k2, "/")
			if (repo == k2 || strings.HasPrefix(repo, k2+"/")) && len(k2) > len(best) {
				best = k2
			}
		}
		if best != "" {
			return f, best, nil
		}
	}
	return "", "", errors.New("no credentials found")
}
