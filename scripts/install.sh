#!/usr/bin/env bash
#
# install.sh - install the Algolia Terraform provider from its GitHub releases.
#
# The provider is not published to the Terraform Registry, so this script fetches
# the release archive with the GitHub CLI (gh) and wires up your Terraform CLI
# config so `terraform` can find it, unless that config already has a
# provider_installation block, in which case it prints what to merge and leaves
# the file alone. Before installing anything, the script authenticates the
# release's SHA256SUMS with the project signing key and verifies the archive.
#
#   Default mode  : filesystem mirror. Behaves like a normal provider - you pin
#                   a version and run `terraform init`.
#   --dev-overrides: quick-test mode. No version pin and no `terraform init`;
#                    Terraform prints a "development overrides" warning.
#
# Requirements: gh (authenticated), gpg, and sha256sum or shasum.
# --dev-overrides also needs unzip.

set -euo pipefail

REPO="algolia/terraform-provider-algolia"
PROVIDER_ADDR="registry.terraform.io/algolia/algolia"
SIGNING_KEY_FINGERPRINT="8A8D999493009BEF83F4A16713B6FAB5E0DBAF30"
SIGNING_KEYSERVER="hkps://keys.openpgp.org"

# --- defaults (all overridable so the script is testable without touching real files) ---
TAG=""
MODE="mirror"
CONFIG_FILE="${TF_CLI_CONFIG_FILE:-$HOME/.terraformrc}"
MIRROR_DIR="$HOME/.terraform.d/plugins"
BIN_DIR="$HOME/.terraform.d/algolia-dev"
TMP=""  # verified release assets; created on demand, removed on exit

err()  { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

usage() {
  cat <<'EOF'
install.sh - install the Algolia Terraform provider from its GitHub releases.

Usage:
  scripts/install.sh [--tag vX.Y.Z] [--dev-overrides] [options]

Options:
  --tag TAG          Release tag to install (default: latest published release,
                     including pre-releases; drafts are skipped)
  --dev-overrides    Use a dev_overrides binary instead of a filesystem mirror
  --config PATH      Terraform CLI config file to update
                     (default: $TF_CLI_CONFIG_FILE or ~/.terraformrc)
  --mirror-dir PATH  Filesystem-mirror base dir (default: ~/.terraform.d/plugins)
  --bin-dir PATH     dev_overrides binary dir (default: ~/.terraform.d/algolia-dev)
  -h, --help         Show this help

Requirements: gh (authenticated), gpg, and sha256sum or shasum;
              unzip is also needed for --dev-overrides.
EOF
  exit 0
}

# --- parse args ---
while [ $# -gt 0 ]; do
  case "$1" in
    --tag)           TAG="${2:?--tag needs a value}"; shift 2 ;;
    --dev-overrides) MODE="dev"; shift ;;
    --config)        CONFIG_FILE="${2:?--config needs a value}"; shift 2 ;;
    --mirror-dir)    MIRROR_DIR="${2:?--mirror-dir needs a value}"; shift 2 ;;
    --bin-dir)       BIN_DIR="${2:?--bin-dir needs a value}"; shift 2 ;;
    -h|--help)       usage ;;
    *)               err "unknown argument: $1 (try --help)" ;;
  esac
done

# --- prerequisite checks ---
command -v gh >/dev/null 2>&1 || err "the GitHub CLI (gh) is required: https://cli.github.com"
command -v gpg >/dev/null 2>&1 || err "gpg is required to verify the release signature"
if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
  err "sha256sum or shasum is required to verify the release archive"
fi
# A token in the environment (e.g. GH_TOKEN in CI) counts as authenticated;
# otherwise fall back to gh's stored credentials.
if [ -z "${GH_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ] && ! gh auth status >/dev/null 2>&1; then
  err "gh is not authenticated - run: gh auth login (or set GH_TOKEN)"
fi

# --- detect platform ---
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) err "unsupported OS '$os' (macOS and Linux only; on Windows follow the README manual steps)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

# --- resolve tag/version ---
if [ -z "$TAG" ]; then
  info "Resolving latest release..."
  # --exclude-drafts matters: goreleaser creates each release as a draft, and gh lists
  # drafts to anyone with write access, so without this a maintainer running the script
  # between a build and its publish would install an unpublished release.
  TAG=$(gh release list --repo "$REPO" --limit 1 --exclude-drafts --json tagName --jq '.[0].tagName' 2>/dev/null) \
    || err "could not list releases for $REPO"
  [ -n "$TAG" ] || err "no releases found for $REPO"
fi
VERSION="${TAG#v}"
ZIP="terraform-provider-algolia_${VERSION}_${os}_${arch}.zip"
SUMS="terraform-provider-algolia_${VERSION}_SHA256SUMS"
SIGNATURE="$SUMS.sig"
info "Installing $REPO $TAG ($os/$arch)"

download_and_verify() {
  TMP=$(mktemp -d)
  trap 'rm -rf "$TMP"' EXIT

  info "Downloading release archive and integrity files..."
  gh release download "$TAG" --repo "$REPO" --pattern "$ZIP" --dir "$TMP" --clobber \
    || err "download failed (is $ZIP an asset of $TAG?)"
  gh release download "$TAG" --repo "$REPO" --pattern "$SUMS" --dir "$TMP" --clobber \
    || err "download failed (is $SUMS an asset of $TAG?)"
  gh release download "$TAG" --repo "$REPO" --pattern "$SIGNATURE" --dir "$TMP" --clobber \
    || err "download failed (is $SIGNATURE an asset of $TAG?)"

  local keyring="$TMP/gnupg"
  mkdir -m 0700 "$keyring"
  gpg --homedir "$keyring" --batch --keyserver "$SIGNING_KEYSERVER" \
    --recv-keys "$SIGNING_KEY_FINGERPRINT" >/dev/null 2>&1 \
    || err "could not retrieve release signing key $SIGNING_KEY_FINGERPRINT"

  local imported_fingerprint
  imported_fingerprint=$(gpg --homedir "$keyring" --batch --with-colons \
    --fingerprint "$SIGNING_KEY_FINGERPRINT" 2>/dev/null \
    | awk -F: '$1 == "fpr" { print $10; exit }')
  [ "$imported_fingerprint" = "$SIGNING_KEY_FINGERPRINT" ] \
    || err "retrieved release signing key has an unexpected fingerprint"

  gpg --homedir "$keyring" --batch --verify "$TMP/$SIGNATURE" "$TMP/$SUMS" >/dev/null 2>&1 \
    || err "signature verification failed for $SUMS"
  info "Checksum signature OK ($SIGNING_KEY_FINGERPRINT)"

  local expected actual
  expected=$(awk -v file="$ZIP" '$2 == file { print $1 }' "$TMP/$SUMS")
  [[ "$expected" =~ ^[[:xdigit:]]{64}$ ]] \
    || err "$SUMS does not contain exactly one valid checksum for $ZIP"
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$TMP/$ZIP")
  else
    actual=$(shasum -a 256 "$TMP/$ZIP")
  fi
  actual=${actual%% *}
  [ "$actual" = "$expected" ] || err "checksum verification failed for $ZIP"
  info "Archive checksum OK"
}

mirror_block() {
  cat <<EOF
provider_installation {
  filesystem_mirror {
    path    = "$MIRROR_DIR"
    include = ["$PROVIDER_ADDR"]
  }
  direct {
    exclude = ["$PROVIDER_ADDR"]
  }
}
EOF
}

dev_block() {
  cat <<EOF
provider_installation {
  dev_overrides {
    "algolia/algolia" = "$BIN_DIR"
  }
  direct {}
}
EOF
}

# Write our provider_installation block into the CLI config, without ever
# clobbering an existing one (Terraform allows only a single such block).
write_config_block() {
  local block="$1"
  if [ ! -e "$CONFIG_FILE" ]; then
    info "Creating $CONFIG_FILE"
    mkdir -p "$(dirname "$CONFIG_FILE")"
    printf '%s\n' "$block" > "$CONFIG_FILE"
    return
  fi
  if grep -q "$PROVIDER_ADDR" "$CONFIG_FILE" 2>/dev/null \
     || grep -q '"algolia/algolia"' "$CONFIG_FILE" 2>/dev/null; then
    info "$CONFIG_FILE already references the Algolia provider - leaving it unchanged."
    return
  fi
  if grep -q 'provider_installation' "$CONFIG_FILE" 2>/dev/null; then
    warn "$CONFIG_FILE already has a provider_installation block."
    warn "Terraform allows only one, so this script will not edit it."
    warn "Merge the following entries into your existing block manually:"
    printf '\n%s\n' "$block" >&2
    return
  fi
  cp "$CONFIG_FILE" "$CONFIG_FILE.bak"
  info "Backed up existing config to $CONFIG_FILE.bak"
  printf '\n%s\n' "$block" >> "$CONFIG_FILE"
  info "Appended provider_installation block to $CONFIG_FILE"
}

install_mirror() {
  local dest="$MIRROR_DIR/$PROVIDER_ADDR"
  info "Installing $ZIP into filesystem mirror: $dest"
  mkdir -p "$dest"
  install -m 0644 "$TMP/$ZIP" "$dest/$ZIP"

  write_config_block "$(mirror_block)"
}

install_dev() {
  command -v unzip >/dev/null 2>&1 || err "unzip is required for --dev-overrides"
  info "Installing binary to $BIN_DIR"
  mkdir -p "$BIN_DIR" "$TMP/unz"
  unzip -o -q "$TMP/$ZIP" -d "$TMP/unz"
  local bin; bin=$(find "$TMP/unz" -type f -name 'terraform-provider-algolia*' | head -1)
  [ -n "$bin" ] || err "provider binary not found inside $ZIP"
  install -m 0755 "$bin" "$BIN_DIR/terraform-provider-algolia"
  write_config_block "$(dev_block)"
}

download_and_verify

if [ "$MODE" = "dev" ]; then
  install_dev
else
  install_mirror
fi

# --- final guidance ---
if [ "$MODE" = "dev" ]; then
  version_line="        # no version constraint needed under dev_overrides"
  next_step="Then run 'terraform plan' (dev_overrides skips 'terraform init')."
else
  version_line="        version = \"$VERSION\""
  next_step="Then run 'terraform init'."
fi

cat <<EOF

Done. Add this to your Terraform configuration:

  terraform {
    required_providers {
      algolia = {
        source  = "algolia/algolia"
$version_line
      }
    }
  }

$next_step
EOF
