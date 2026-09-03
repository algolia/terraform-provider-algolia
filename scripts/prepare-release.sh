#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || ! "$1" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "usage: $0 <major.minor.patch>" >&2
  exit 1
fi

repo_root=$(cd "$(dirname "$0")/.." && pwd)
version="$1"
release_date=$(date '+%B %e, %Y' | sed 's/  / /g')
previous_version=$(
  git -C "$repo_root" tag --list 'v*' --sort=-version:refname |
    awk '/^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/ {
      sub(/^v/, "")
      print
      exit
    }'
)

version_is_greater() {
  local candidate_major candidate_minor candidate_patch current_major current_minor current_patch
  IFS=. read -r candidate_major candidate_minor candidate_patch <<<"$1"
  IFS=. read -r current_major current_minor current_patch <<<"$2"
  ((candidate_major > current_major)) ||
    ((candidate_major == current_major && candidate_minor > current_minor)) ||
    ((candidate_major == current_major && candidate_minor == current_minor && candidate_patch > current_patch))
}

if [[ -z "$previous_version" ]]; then
  echo "no stable release tag found; fetch tags before preparing a release" >&2
  exit 1
fi
if ! version_is_greater "$version" "$previous_version"; then
  echo "$version must be greater than the current release $previous_version" >&2
  exit 1
fi
if [[ $(sed -n '1p' "$repo_root/CHANGELOG.md") != "## Unreleased" ]]; then
  echo "CHANGELOG.md must start with ## Unreleased" >&2
  exit 1
fi
if grep -q "^## $version " "$repo_root/CHANGELOG.md"; then
  echo "CHANGELOG.md already contains release $version" >&2
  exit 1
fi
if git -C "$repo_root" rev-parse --quiet --verify "refs/tags/v$version" >/dev/null; then
  echo "tag v$version already exists" >&2
  exit 1
fi
if [[ -n $(git -C "$repo_root" status --porcelain) ]]; then
  echo "release preparation requires a clean worktree" >&2
  exit 1
fi

version_files=(
  README.md
  INSTALL.md
  examples/provider/provider.tf
  examples/ecommerce-search/versions.tf
  examples/ingestion-pipeline/versions.tf
  examples/media-search/versions.tf
  skills/algolia-terraform-provider/SKILL.md
)

for relative_path in "${version_files[@]}"; do
  if ! grep -Eq "^[[:space:]]*version[[:space:]]*=[[:space:]]*\"${previous_version//./\\.}\"" \
    "$repo_root/$relative_path"; then
    echo "$relative_path does not pin the current release $previous_version" >&2
    exit 1
  fi
done

rollback() {
  git -C "$repo_root" restore -- CHANGELOG.md docs "${version_files[@]}"
  rm -f "$repo_root/CHANGELOG.md.tmp"
}
rollback_on_failure() {
  local status=$?
  if ((status != 0)); then
    rollback
  fi
}
trap rollback_on_failure EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for relative_path in "${version_files[@]}"; do
  file="$repo_root/$relative_path"
  sed -i.bak "s/${previous_version//./\\.}/$version/g" "$file"
  rm "$file.bak"
done

changelog_tmp="$repo_root/CHANGELOG.md.tmp"
awk -v heading="## $version ($release_date)" 'NR == 1 { print heading; next } { print }' \
  "$repo_root/CHANGELOG.md" >"$changelog_tmp"
mv "$changelog_tmp" "$repo_root/CHANGELOG.md"

make -C "$repo_root" generate

absolute_version_files=()
for relative_path in "${version_files[@]}"; do
  absolute_version_files+=("$repo_root/$relative_path")
done
if grep -n -F "$previous_version" "${absolute_version_files[@]}" >/dev/null; then
  echo "a release-version reference was not updated" >&2
  exit 1
fi
if ! grep -Eq "^[[:space:]]*version[[:space:]]*=[[:space:]]*\"${version//./\\.}\"" \
  "$repo_root/docs/index.md"; then
  echo "generated provider documentation does not reference $version" >&2
  exit 1
fi
# A literal sweep missed media-search in two earlier releases because its provider
# constraint used `~> 0.1`. Inspect every Terraform file that declares this provider,
# including examples added after this script, and require an exact prepared pin.
while IFS= read -r -d '' terraform_file; do
  if grep -Eq 'source[[:space:]]*=[[:space:]]*"algolia/algolia"' "$terraform_file" &&
    ! grep -Eq "^[[:space:]]*version[[:space:]]*=[[:space:]]*\"${version//./\\.}\"" "$terraform_file"; then
    echo "$terraform_file does not pin algolia/algolia to $version" >&2
    exit 1
  fi
done < <(find "$repo_root/examples" -name '*.tf' -print0)
remaining_references=$(git -C "$repo_root" grep -l -F "$previous_version" -- \
  README.md INSTALL.md AGENTS.md docs examples skills ':!CHANGELOG.md' || true)
if [[ -n "$remaining_references" ]]; then
  echo "the previous release version remains outside CHANGELOG.md:" >&2
  echo "$remaining_references" >&2
  exit 1
fi

trap - EXIT INT TERM

echo "prepared release $version from $previous_version; review the diff before committing"
