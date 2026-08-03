# Installing the provider

This provider is not published to the Terraform Registry, so Terraform will not fetch it
for you. Instead you place the release archive on disk and point Terraform at it with a
[filesystem mirror](https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror).
No private registry is required.

Terraform cannot install a provider from a git URL. `source = "git::https://..."` works for
modules, not providers, so one of the routes below is necessary.

## Quickest, no checkout

Requires [`gh`](https://cli.github.com/) authenticated (`gh auth login`), which the
installer needs anyway to download the release:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

That resolves the latest release, verifies its checksum, populates the mirror, and writes a
`provider_installation` block to `~/.terraformrc`. It prints the version to pin when it
finishes.

## From a checkout

```bash
scripts/install.sh                 # latest release
scripts/install.sh --tag v0.1.0    # a specific release
```

Useful options: `--config PATH` and `--mirror-dir PATH` to write somewhere other than
`~/.terraformrc` and `~/.terraform.d/plugins`, and `--dev-overrides` to install a locally
built binary through `dev_overrides` instead of a mirror. `--dev-overrides` skips version
pinning and `terraform init` entirely, which suits a quick look at a resource and does not
suit anything holding state.

The script will not overwrite an existing `provider_installation` block, since Terraform
allows only one. If you already have one it prints the entries to merge and changes nothing.

## By hand

The script automates exactly these steps.

**1. Download the archive for your platform** from the
[releases page](https://github.com/algolia/terraform-provider-algolia/releases):
`terraform-provider-algolia_<version>_<os>_<arch>.zip`, where `<os>_<arch>` is one of
`darwin_arm64`, `darwin_amd64`, `linux_amd64`, `linux_arm64`, `windows_amd64` or
`windows_arm64`.

**2. Put it in the mirror without unzipping it.** The mirror uses the packed layout:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia
mv ~/Downloads/terraform-provider-algolia_0.1.0_darwin_arm64.zip \
   ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia/
```

On Windows the directory is
`%APPDATA%\terraform.d\plugins\registry.terraform.io\algolia\algolia\`.

**3. Point Terraform at the mirror** from your CLI config, `~/.terraformrc` on macOS and
Linux or `%APPDATA%\terraform.rc` on Windows. This keeps every other provider coming from
the public registry:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/Users/<you>/.terraform.d/plugins"
    include = ["registry.terraform.io/algolia/algolia"]
  }
  direct {
    exclude = ["registry.terraform.io/algolia/algolia"]
  }
}
```

**4. Optionally verify the download.** Each release ships a GPG-signed `SHA256SUMS` and
`SHA256SUMS.sig`. Terraform does not check signatures for filesystem-mirror installs, so
verify by hand if you want the assurance:

```bash
shasum -a 256 -c terraform-provider-algolia_0.1.0_SHA256SUMS 2>&1 | grep OK
```

## Pin the version you installed

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.0"
    }
  }
}
```

A mirror holds exactly the versions you have put in it. A constraint that resolves to
something absent fails rather than fetching it:

```
Error: Failed to query available provider packages
algolia/algolia: no available releases match the given constraints
```

If you see that, either the mirror does not have the version you asked for or the
constraint does not match what is there. `install.sh` prints the version it installed, and
the archive filename in the mirror directory is the other way to check.

## When a valid configuration is rejected

An error naming a resource that exists, such as
`The provider algolia/algolia does not support resource type "algolia_rule"`, or an
`Unsupported argument` for an argument that is documented, usually means Terraform loaded a
different build than you think.

`~/.terraform.d/plugins` is also Terraform's implicit local plugin directory, and it accepts
an unpacked layout (`<version>/<os>_<arch>/terraform-provider-algolia`) alongside the packed
`.zip` this installer writes. An old unpacked build left there from `go build` shadows the
release archive of the same version number, so `terraform init` reports the version you
pinned while running older code. Check for one and remove it:

```bash
ls ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia/
```

A directory named after a version holds an unpacked build; a `.zip` is what the installer
put there.
