#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "${script_dir}/../.." && pwd)"
workspace_root="$(cd "${project_root}/../.." && pwd)"
state_dir="${PROOFLAYER_LAB_STATE_DIR:-${workspace_root}/work/prooflayer-lab}"
splunk_env="${project_root}/deployments/splunk/.env"
day30_env="${script_dir}/day30.env"
certificate="${state_dir}/day30-control-plane.pem"
private_key="${state_dir}/day30-control-plane.key"
go_cache="${PROOFLAYER_GO_CACHE:-/tmp/prooflayer-go-cache}"

command -v openssl >/dev/null 2>&1 || {
  echo "openssl is required to prepare the Day 30 lab." >&2
  exit 1
}
command -v go >/dev/null 2>&1 || {
  echo "Go is required to prepare the Day 30 lab." >&2
  exit 1
}
[[ -f "${splunk_env}" ]] || {
  echo "deployments/splunk/.env is required; configure HEC and the Observer first." >&2
  exit 1
}

set -a
if [[ -f "${day30_env}" ]]; then
  source "${day30_env}"
fi
source "${splunk_env}"
set +a

[[ -n "${PROOFLAYER_HEC_TOKEN:-}" && -n "${PROOFLAYER_OBSERVER_PASSWORD:-}" ]] || {
  echo "HEC and Observer credentials are missing from deployments/splunk/.env." >&2
  exit 1
}

mkdir -p "${state_dir}" "${go_cache}"
PROOFLAYER_RUNNER_TOKEN="${PROOFLAYER_RUNNER_TOKEN:-$(openssl rand -hex 32)}"
PROOFLAYER_SIGNING_SEED="${PROOFLAYER_SIGNING_SEED:-$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n')}"
export PROOFLAYER_SIGNING_SEED
PROOFLAYER_SIGNING_PUBLIC_KEY="$(
  cd "${project_root}/control-plane"
  GOCACHE="${go_cache}" go run ./cmd/prooflayer-signing-public-key
)"

if [[ ! -f "${certificate}" || ! -f "${private_key}" ]]; then
  openssl req -x509 -newkey rsa:2048 -sha256 -nodes \
    -keyout "${private_key}" \
    -out "${certificate}" \
    -days 30 \
    -subj "/CN=ProofLayer Day 30 Isolated Lab" \
    -addext "subjectAltName=IP:127.0.0.1,IP:10.0.2.100" \
    >/dev/null 2>&1
  chmod 600 "${certificate}" "${private_key}"
fi

temporary_env="${day30_env}.tmp"
umask 077
{
  printf 'PROOFLAYER_RUNNER_TOKEN=%s\n' "${PROOFLAYER_RUNNER_TOKEN}"
  printf 'PROOFLAYER_SIGNING_SEED=%s\n' "${PROOFLAYER_SIGNING_SEED}"
  printf 'PROOFLAYER_SIGNING_PUBLIC_KEY=%s\n' "${PROOFLAYER_SIGNING_PUBLIC_KEY}"
  printf 'PROOFLAYER_HEC_TOKEN=%s\n' "${PROOFLAYER_HEC_TOKEN}"
  printf 'PROOFLAYER_OBSERVER_PASSWORD=%s\n' "${PROOFLAYER_OBSERVER_PASSWORD}"
  printf 'PROOFLAYER_RUNNER_TLS_CERT=%s\n' "${certificate}"
  printf 'PROOFLAYER_RUNNER_TLS_KEY=%s\n' "${private_key}"
} > "${temporary_env}"
mv "${temporary_env}" "${day30_env}"
chmod 600 "${day30_env}"

echo "Prepared ignored Day 30 secrets and the isolated-lab TLS certificate."
echo "Next: source deployments/lab/day30.env before starting the Control Plane or building Runner media."
