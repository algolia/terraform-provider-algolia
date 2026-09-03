# Installing the provider

Terraform installs the provider directly from the public
[Terraform Registry](https://registry.terraform.io/providers/algolia/algolia/latest). Declare
the provider in the configuration:

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

Then initialize the working directory:

```bash
terraform init
```

Terraform will select the pinned Registry release and record it and its checksums in
`.terraform.lock.hcl`. Commit that lock file so subsequent runs select and verify the same
provider release. To move to a newer release, update `version` in `required_providers`, then
run:

```shell
terraform init -upgrade
```

### Migrating from the release installer

If you previously used `install.sh` and now want to install from the Registry, its
`provider_installation` block excludes Algolia from direct installation. Replace the block the
installer created in `~/.terraformrc` (or the configured CLI configuration file) with:

```hcl
provider_installation {
  direct {}
}
```

Then run `terraform init -upgrade`. Do not only delete the block while the provider remains in
the default mirror at `~/.terraform.d/plugins`: without an explicit block, Terraform's
[implied installation configuration](https://developer.hashicorp.com/terraform/cli/config/config-file#implied-local-mirror-directories)
detects providers there and automatically excludes them from direct installation. To return to
that implicit configuration instead, remove both the block and
`~/.terraform.d/plugins/registry.terraform.io/algolia/algolia`.

## Install from a release archive

Use the release installer when direct Registry access is unavailable or when testing a
specific GitHub release through a local Terraform installation mirror. It supports macOS and
Linux, including WSL when Terraform runs inside WSL, but not native Windows installations.

The installer requires [`gh`](https://cli.github.com/) authenticated (`gh auth login`), GnuPG,
and either `sha256sum` or `shasum`. Without a checkout:

```bash
gh api -H "Accept: application/vnd.github.raw" \
  repos/algolia/terraform-provider-algolia/contents/scripts/install.sh | bash
```

From a checkout:

```bash
scripts/install.sh                 # latest published release
scripts/install.sh --tag vX.Y.Z    # a specific release
```

It resolves the selected release, populates a
[filesystem mirror](https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror),
configures Terraform to use it when safe, and prints the exact version to declare.

"Most recent" includes pre-releases. Before writing the archive or provider binary, the
installer retrieves the project signing key by its pinned fingerprint, verifies the signature
on the release's `SHA256SUMS`, and verifies the selected archive against that authenticated
manifest. It stops without installing when any download, tool, signature, or checksum check
fails.

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
`.zip` this installer writes. An old unpacked build left there from `go build` can shadow a
Registry or release-archive package of the same version, so `terraform init` reports the
version you pinned while running older code. Check for one and remove it:

```bash
ls ~/.terraform.d/plugins/registry.terraform.io/algolia/algolia/
```

A directory named after a version holds an unpacked build; a `.zip` is what the installer
put there.
