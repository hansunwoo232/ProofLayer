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
  GOCACHE="${go_cache}" GOOS=windows GOARCH=arm64 go build \
    -trimpath \
    -o "${staging_dir}/prooflayer-day30-lab.exe" \
    ./cmd/prooflayer-day30-lab
)

cp "${script_dir}/windows/run-runner-day17.ps1" "${staging_dir}/"
cp "${script_dir}/windows/run-registry-day24.ps1" "${staging_dir}/"
cp "${script_dir}/windows/run-scheduled-task-day25.ps1" "${staging_dir}/"
cp "${script_dir}/windows/run-day30.ps1" "${staging_dir}/"

day30_variables=(
  PROOFLAYER_RUNNER_TOKEN
  PROOFLAYER_SIGNING_PUBLIC_KEY
  PROOFLAYER_HEC_TOKEN
  PROOFLAYER_OBSERVER_PASSWORD
)
configured_day30=0
for variable in "${day30_variables[@]}"; do
  [[ -n "${!variable:-}" ]] && configured_day30=$((configured_day30 + 1))
done
if (( configured_day30 != 0 && configured_day30 != ${#day30_variables[@]} )); then
  echo "All Day 30 secret variables must be set together." >&2
  exit 1
fi
if (( configured_day30 == ${#day30_variables[@]} )); then
  command -v python3 >/dev/null 2>&1 || {
    echo "python3 is required to create the local Day 30 config." >&2
    exit 1
  }
  PROOFLAYER_DAY30_CONFIG_PATH="${staging_dir}/prooflayer-day30-config.json" \
    python3 -c 'import json, os; p=os.environ["PROOFLAYER_DAY30_CONFIG_PATH"]; d={"schema_version":"1.0","control_plane_base_url":"https://10.0.2.100:8788","runner_token":os.environ["PROOFLAYER_RUNNER_TOKEN"],"signing_public_key":os.environ["PROOFLAYER_SIGNING_PUBLIC_KEY"],"hec_endpoint":"https://10.0.2.100:8088/services/collector/event","hec_token":os.environ["PROOFLAYER_HEC_TOKEN"],"splunk_base_url":"https://10.0.2.100:8089","splunk_observer_password":os.environ["PROOFLAYER_OBSERVER_PASSWORD"],"allow_lab_self_signed_tls":True}; open(p,"w",encoding="utf-8").write(json.dumps(d,separators=(",",":")))'
  chmod 600 "${staging_dir}/prooflayer-day30-config.json"
fi
shasum -a 256 \
  "${staging_dir}/prooflayer-runner-lab.exe" \
  "${staging_dir}/prooflayer-registry-canary-lab.exe" \
  "${staging_dir}/prooflayer-scheduled-task-canary-lab.exe" \
  "${staging_dir}/prooflayer-day30-lab.exe" \
  > "${staging_dir}/SHA256SUMS.txt"

rm -f "${temporary_iso}"
trap 'rm -f "${temporary_iso}"' EXIT
hdiutil makehybrid -iso -joliet -o "${temporary_iso}" "${staging_dir}" >/dev/null
mv "${temporary_iso}" "${output_iso}"
trap - EXIT

echo "Created local Runner media: ${output_iso}"
echo "SHA-256: $(shasum -a 256 "${output_iso}" | awk '{print $1}')"
