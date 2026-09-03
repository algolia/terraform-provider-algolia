//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireInstallScript skips unless this is an e2e run (TF_ACC) and the tools
// the install script needs are present: bash, gh (authenticated), gpg, a SHA-256
// tool, terraform, and unzip. Unlike the other e2e tests it does NOT need Algolia
// credentials - scripts/install.sh only downloads the published provider release
// and runs terraform init/plan; it makes no live Algolia API calls.
func requireInstallScript(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("install-script test is skipped unless TF_ACC is set; run `make e2e`")
	}
	for _, tool := range []string{"bash", "gh", "gpg", "terraform", "unzip"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%q not found on PATH; skipping install-script test", tool)
		}
	}
	if _, err := exec.LookPath("sha256sum"); err != nil {
		if _, err := exec.LookPath("shasum"); err != nil {
			t.Skip("sha256sum and shasum not found on PATH; skipping install-script test")
		}
	}
	// A token in the environment (GH_TOKEN in CI) counts as authenticated;
	// otherwise fall back to gh's stored credentials.
	if os.Getenv("GH_TOKEN") == "" && os.Getenv("GITHUB_TOKEN") == "" {
		if err := exec.Command("gh", "auth", "status").Run(); err != nil {
			t.Skip("gh is not authenticated (set GH_TOKEN or run `gh auth login`); skipping install-script test")
		}
	}
}

// repoRoot returns the repository root. `go test` runs with the working
// directory set to the package dir (internal/e2e), so the root is two levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// runCmd executes name with args in dir, appending extraEnv to the current
// environment, and fails the test on a non-zero exit, surfacing the output.
func runCmd(t *testing.T, dir string, extraEnv []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

func latestReleaseTag(t *testing.T, root string) string {
	t.Helper()
	out := runCmd(t, root, nil, "gh", "release", "list",
		"--repo", "algolia/terraform-provider-algolia",
		"--limit", "1", "--json", "tagName", "--jq", ".[0].tagName")
	tag := strings.TrimSpace(out)
	if tag == "" {
		t.Fatal("no releases found for algolia/terraform-provider-algolia")
	}
	return tag
}

// TestE2EInstallScript exercises the signed release installer (scripts/install.sh)
// against the latest published release in both install modes, proving it works
// whenever direct Registry access is unavailable.
func TestE2EInstallScript(t *testing.T) {
	requireInstallScript(t)

	root := repoRoot(t)
	script := filepath.Join(root, "scripts", "install.sh")
	tag := latestReleaseTag(t, root)
	version := strings.TrimPrefix(tag, "v")
	t.Logf("testing quick install of %s", tag)

	// Filesystem mirror: install, then `terraform init` must resolve the
	// provider from the mirror (the normal, versioned consumption path).
	t.Run("filesystem_mirror", func(t *testing.T) {
		home := t.TempDir()
		cfg := filepath.Join(home, "terraformrc")
		mirror := filepath.Join(home, "mirror")

		runCmd(t, root, nil, "bash", script, "--tag", tag, "--config", cfg, "--mirror-dir", mirror)

		zips, _ := filepath.Glob(filepath.Join(mirror, "registry.terraform.io", "algolia", "algolia", "*.zip"))
		if len(zips) == 0 {
			t.Fatalf("no provider archive found in mirror %s", mirror)
		}

		tfDir := filepath.Join(home, "tf")
		if err := os.MkdirAll(tfDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, tfDir, mirrorConfig(version))

		out := runCmd(t, tfDir, []string{"TF_CLI_CONFIG_FILE=" + cfg}, "terraform", "init", "-input=false")
		if !strings.Contains(out, "algolia/algolia") {
			t.Fatalf("terraform init did not install the provider from the mirror:\n%s", out)
		}
	})

	// dev_overrides: install the binary, then `terraform plan` (no init) must
	// load it and produce a create plan.
	t.Run("dev_overrides", func(t *testing.T) {
		home := t.TempDir()
		cfg := filepath.Join(home, "terraformrc")
		bin := filepath.Join(home, "bin")

		runCmd(t, root, nil, "bash", script, "--dev-overrides", "--tag", tag, "--config", cfg, "--bin-dir", bin)

		info, err := os.Stat(filepath.Join(bin, "terraform-provider-algolia"))
		if err != nil {
			t.Fatalf("provider binary not installed: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatal("installed provider binary is not executable")
		}

		tfDir := filepath.Join(home, "tf")
		if err := os.MkdirAll(tfDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, tfDir, devOverrideConfig())

		out := runCmd(t, tfDir, []string{"TF_CLI_CONFIG_FILE=" + cfg}, "terraform", "plan", "-input=false")
		if !strings.Contains(out, "1 to add") {
			t.Fatalf("expected a create plan from the dev_overrides binary, got:\n%s", out)
		}
	})
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing main.tf: %v", err)
	}
}

func mirrorConfig(version string) string {
	return fmt.Sprintf(`terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = %q
    }
  }
}
`, version)
}

func devOverrideConfig() string {
	return `terraform {
  required_providers {
    algolia = {
      source = "algolia/algolia"
    }
  }
}

provider "algolia" {
  app_id  = "citestapp"
  api_key = "ci-test-key"
}

resource "algolia_index" "smoke" {
  name                = "ci-quick-install-smoke"
  deletion_protection = false
}
`
}
