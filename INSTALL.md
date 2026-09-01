# Installing the provider

This provider is not published to the Terraform Registry, so Terraform will not fetch it
for you. Instead you place the release archive on disk and point Terraform at it with a
[filesystem mirror](https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror).
No private registry is required.

Terraform cannot install a provider from a git URL. `source = "git::https://..."` works for
modules, not providers, so one of the routes below is necessary.

The installer supports macOS and Linux, including WSL when Terraform runs inside WSL. It does
not support native Windows installations.

## Quickest, no checkout

Requires [`gh`](https://cli.github.com/) authenticated (`gh auth login`), GnuPG, and either
`sha256sum` or `shasum`:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

That resolves the most recent release, populates the mirror, and writes a
`provider_installation` block to `~/.terraformrc` unless one is already there. It prints the
version to pin when it finishes.

"Most recent" includes pre-releases. Before writing the archive or provider binary, the
installer retrieves the project signing key by its pinned fingerprint, verifies the signature
on the release's `SHA256SUMS`, and verifies the selected archive against that authenticated
manifest. It stops without installing when any download, tool, signature, or checksum check
fails.

## From a checkout

```bash
scripts/install.sh                 # latest release
scripts/install.sh --tag v0.1.2    # a specific release
```

Useful options: `--config PATH` and `--mirror-dir PATH` to write somewhere other than
`~/.terraformrc` and `~/.terraform.d/plugins`, `--bin-dir PATH` for the `dev_overrides`
binary, and `--dev-overrides` to wire the provider up through `dev_overrides` instead of a
mirror. `--dev-overrides` still installs the released binary, downloading and unpacking the
same archive; what it changes is that it skips version pinning and `terraform init` entirely,
which suits a quick look at a resource and does not suit anything holding state. To run a
binary you built yourself, point `dev_overrides` at it directly rather than using this script.

The script never edits an existing `provider_installation` block, since Terraform allows only
one. What it does depends on what is already there: if the config already mentions the Algolia
provider it leaves the file untouched and says so, and if there is a `provider_installation`
block that does not mention Algolia it prints the entries for you to merge in by hand.

## Pin the version you installed

```hcl
terraform {
  required_providers {
    algolia = {
      source  = "algolia/algolia"
      version = "0.1.2"
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
