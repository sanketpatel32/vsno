package precheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCIDRConflicts(t *testing.T) {
	guest := []string{"10.0.2.0/24", "10.132.0.0/14", "172.31.0.0/16"}

	// Default OpenShift host networks: no conflict with the vSNO guest defaults.
	got, err := FindCIDRConflicts(guest, []string{"10.128.0.0/14", "172.30.0.0/16"})
	if err != nil || len(got) != 0 {
		t.Fatalf("want no conflicts, got %v err=%v", got, err)
	}

	// A host using a /12 service network swallows the guest service network.
	got, err = FindCIDRConflicts(guest, []string{"10.128.0.0/14", "172.16.0.0/12"})
	if err != nil || len(got) != 1 || got[0].Guest != "172.31.0.0/16" {
		t.Fatalf("want 172.31.0.0/16 conflict, got %v err=%v", got, err)
	}

	if _, err := FindCIDRConflicts([]string{"bogus"}, []string{"10.0.0.0/8"}); err == nil {
		t.Fatal("want error for invalid CIDR")
	}
}

func TestFindRegistryAuth(t *testing.T) {
	dir := t.TempDir()
	podman := filepath.Join(dir, "auth.json")
	docker := filepath.Join(dir, "config.json")
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(podman, []byte(`{"auths":{"quay.io":{"auth":"aaa"},"quay.io/me":{"auth":"bbb"},"registry.redhat.io":{}}}`), 0o600))
	must(os.WriteFile(docker, []byte(`{"auths":{"https://ghcr.io":{"auth":"ccc"}}}`), 0o600))
	files := []string{podman, docker}

	cases := []struct {
		repo, wantKey, wantFile string
	}{
		{"quay.io/me/ibu-seed:4.20", "quay.io/me", podman},
		{"quay.io/other/ibu-seed", "quay.io", podman},
		{"docker://quay.io/me/ibu-seed", "quay.io/me", podman},
		{"ghcr.io/me/seed", "ghcr.io", docker},
	}
	for _, c := range cases {
		f, k, err := FindRegistryAuth(c.repo, files)
		if err != nil || k != c.wantKey || f != c.wantFile {
			t.Errorf("%s: got (%s,%s,%v), want (%s,%s)", c.repo, f, k, err, c.wantFile, c.wantKey)
		}
	}

	// A key with no inline auth (credential helper) does not count.
	if _, _, err := FindRegistryAuth("registry.redhat.io/foo/bar", files); err == nil {
		t.Error("registry.redhat.io has no inline auth; want error")
	}
	// quay.io/me must not match quay.io/meow.
	if _, k, _ := FindRegistryAuth("quay.io/meow/x", files); k != "quay.io" {
		t.Errorf("quay.io/meow matched %q, want quay.io", k)
	}
}
