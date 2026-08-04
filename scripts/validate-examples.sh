#!/usr/bin/env bash

set -euo pipefail

if ! command -v terraform >/dev/null 2>&1; then
  echo "terraform is required to validate examples" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/terraform-provider-algolia-examples.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

plugin_dir="$work_dir/plugins"
mkdir -p "$plugin_dir"
go build -o "$plugin_dir/terraform-provider-algolia" "$repo_root"

cli_config="$work_dir/terraformrc"
cat >"$cli_config" <<EOF
provider_installation {
  dev_overrides {
    "algolia/algolia" = "$plugin_dir"
  }
  direct {}
}
EOF

terraform -chdir="$repo_root" fmt -check -recursive examples

validated=0
while IFS= read -r example_dir; do
  relative_dir="${example_dir#"$repo_root"/}"
  validate_dir="$work_dir/configs/$relative_dir"
  mkdir -p "$validate_dir"
  cp "$example_dir"/*.tf "$validate_dir"/

  # tfplugindocs resource/data-source snippets intentionally omit provider
  # installation boilerplate. Inject the real registry address in the temporary
  # copy rather than teaching Terraform that the nonexistent hashicorp/algolia
  # address is valid.
  if ! grep -Rqs 'required_providers' "$validate_dir"; then
    cat >"$validate_dir/provider_source.tf" <<'EOF'
terraform {
  required_providers {
    algolia = {
      source = "algolia/algolia"
    }
  }
}
EOF
  fi

  if [[ -f "$example_dir/import.sh" ]]; then
    while IFS= read -r resource_address; do
      resource_type="${resource_address%%.*}"
      resource_name="${resource_address#*.}"
      if ! grep -Eq "resource[[:space:]]+\"${resource_type}\"[[:space:]]+\"${resource_name}\"" "$example_dir"/*.tf; then
        echo "$relative_dir/import.sh references undeclared resource $resource_address" >&2
        exit 1
      fi
    done < <(awk '$1 == "terraform" && $2 == "import" { print $3 }' "$example_dir/import.sh")
  fi

  echo "Validating $relative_dir"
  TF_CLI_CONFIG_FILE="$cli_config" terraform -chdir="$validate_dir" validate -no-color >/dev/null
  validated=$((validated + 1))
done < <(find "$repo_root/examples" -type f -name '*.tf' -exec dirname {} \; | sort -u)

if ((validated == 0)); then
  echo "No Terraform examples were found." >&2
  exit 1
fi

echo "Validated $validated example directories."
