#!/usr/bin/env bash
#
# install.sh - install the Algolia Terraform provider from internal GitHub releases.
#
# While the provider is internal-only (not on the public Terraform Registry),
# this script fetches the signed release artifact with the GitHub CLI (gh) and
# wires up your Terraform CLI config so `terraform` can find it.
#
#   Default mode  : filesystem mirror. Behaves like a normal provider - you pin
#                   a version and run `terraform init`.
#   --dev-overrides: quick-test mode. No version pin and no `terraform init`;
#                    Terraform prints a "development overrides" warning.
#
# Requirements: gh (authenticated). --dev-overrides also needs unzip.

set -euo pipefail

REPO="algolia/terraform-provider-algolia"
PROVIDER_ADDR="registry.terraform.io/algolia/algolia"

# --- defaults (all overridable so the script is testable without touching real files) ---
TAG=""
MODE="mirror"
CONFIG_FILE="${TF_CLI_CONFIG_FILE:-$HOME/.terraformrc}"
MIRROR_DIR="$HOME/.terraform.d/plugins"
BIN_DIR="$HOME/.terraform.d/algolia-dev"

err()  { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

usage() {
  cat <<'EOF'
install.sh - install the Algolia Terraform provider from internal GitHub releases.

Usage:
  scripts/install.sh [--tag vX.Y.Z] [--dev-overrides] [options]

Options:
  --tag TAG          Release tag to install (default: latest, including pre-releases)
  --dev-overrides    Use a dev_overrides binary instead of a filesystem mirror
  --config PATH      Terraform CLI config file to update
                     (default: $TF_CLI_CONFIG_FILE or ~/.terraformrc)
  --mirror-dir PATH  Filesystem-mirror base dir (default: ~/.terraform.d/plugins)
  --bin-dir PATH     dev_overrides binary dir (default: ~/.terraform.d/algolia-dev)
  -h, --help         Show this help

Requirements: gh (authenticated); unzip is also needed for --dev-overrides.
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
gh auth status >/dev/null 2>&1 || err "gh is not authenticated - run: gh auth login"

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
  TAG=$(gh release list --repo "$REPO" --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null) \
    || err "could not list releases for $REPO"
  [ -n "$TAG" ] || err "no releases found for $REPO"
fi
VERSION="${TAG#v}"
ZIP="terraform-provider-algolia_${VERSION}_${os}_${arch}.zip"
info "Installing $REPO $TAG ($os/$arch)"

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
  info "Downloading $ZIP into filesystem mirror: $dest"
  mkdir -p "$dest"
  gh release download "$TAG" --repo "$REPO" --pattern "$ZIP" --dir "$dest" --clobber \
    || err "download failed (is $ZIP an asset of $TAG?)"

  # Best-effort integrity check against the published SHA256SUMS (not GPG).
  if gh release download "$TAG" --repo "$REPO" --pattern '*_SHA256SUMS' --dir "$dest" --clobber >/dev/null 2>&1; then
    local sums; sums=$(ls "$dest"/*_SHA256SUMS 2>/dev/null | head -1 || true)
    if [ -n "$sums" ]; then
      local checker=""
      command -v shasum   >/dev/null 2>&1 && checker="shasum -a 256 -c"
      command -v sha256sum >/dev/null 2>&1 && checker="sha256sum -c"
      if [ -n "$checker" ]; then
        ( cd "$dest" && grep " $ZIP\$" "$(basename "$sums")" | $checker - >/dev/null 2>&1 ) \
          && info "Checksum OK" || warn "checksum verification skipped/failed for $ZIP"
      fi
      rm -f "$sums"
    fi
  fi

  write_config_block "$(mirror_block)"
}

install_dev() {
  command -v unzip >/dev/null 2>&1 || err "unzip is required for --dev-overrides"
  local tmp; tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  info "Downloading $ZIP..."
  gh release download "$TAG" --repo "$REPO" --pattern "$ZIP" --dir "$tmp" --clobber \
    || err "download failed (is $ZIP an asset of $TAG?)"
  info "Installing binary to $BIN_DIR"
  mkdir -p "$BIN_DIR" "$tmp/unz"
  unzip -o -q "$tmp/$ZIP" -d "$tmp/unz"
  local bin; bin=$(find "$tmp/unz" -type f -name 'terraform-provider-algolia*' | head -1)
  [ -n "$bin" ] || err "provider binary not found inside $ZIP"
  install -m 0755 "$bin" "$BIN_DIR/terraform-provider-algolia"
  write_config_block "$(dev_block)"
}

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
