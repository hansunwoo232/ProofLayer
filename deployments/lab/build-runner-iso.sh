#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "${script_dir}/../.." && pwd)"
workspace_root="$(cd "${project_root}/../.." && pwd)"
state_dir="${PROOFLAYER_LAB_STATE_DIR:-${workspace_root}/work/prooflayer-lab}"
staging_dir="${state_dir}/runner-tools-iso"
output_iso="${state_dir}/prooflayer-runner.iso"
temporary_iso="${state_dir}/prooflayer-runner.$$.iso"
go_cache="${PROOFLAYER_GO_CACHE:-/tmp/prooflayer-go-cache}"

command -v go >/dev/null 2>&1 || {
  echo "Go is required to build the Runner lab media." >&2
  exit 1
}
command -v hdiutil >/dev/null 2>&1 || {
  echo "hdiutil is required to build the Runner lab media on macOS." >&2
  exit 1
}

rm -rf "${staging_dir}"
mkdir -p "${staging_dir}" "${go_cache}"

(
  cd "${project_root}/runner"
  GOCACHE="${go_cache}" GOOS=windows GOARCH=arm64 go build \
    -trimpath \
    -o "${staging_dir}/prooflayer-runner-lab.exe" \
    ./cmd/prooflayer-runner-lab
  GOCACHE="${go_cache}" GOOS=windows GOARCH=arm64 go build \
    -trimpath \
    -o "${staging_dir}/prooflayer-registry-canary-lab.exe" \
    ./cmd/prooflayer-registry-canary-lab
  GOCACHE="${go_cache}" GOOS=windows GOARCH=arm64 go build \
    -trimpath \
    -o "${staging_dir}/prooflayer-scheduled-task-canary-lab.exe" \
    ./cmd/prooflayer-scheduled-task-canary-lab
)

cp "${script_dir}/windows/run-runner-day17.ps1" "${staging_dir}/"
cp "${script_dir}/windows/run-registry-day24.ps1" "${staging_dir}/"
cp "${script_dir}/windows/run-scheduled-task-day25.ps1" "${staging_dir}/"
shasum -a 256 \
  "${staging_dir}/prooflayer-runner-lab.exe" \
  "${staging_dir}/prooflayer-registry-canary-lab.exe" \
  "${staging_dir}/prooflayer-scheduled-task-canary-lab.exe" \
  > "${staging_dir}/SHA256SUMS.txt"

rm -f "${temporary_iso}"
trap 'rm -f "${temporary_iso}"' EXIT
hdiutil makehybrid -iso -joliet -o "${temporary_iso}" "${staging_dir}" >/dev/null
mv "${temporary_iso}" "${output_iso}"
trap - EXIT

echo "Created local Runner media: ${output_iso}"
echo "SHA-256: $(shasum -a 256 "${output_iso}" | awk '{print $1}')"
