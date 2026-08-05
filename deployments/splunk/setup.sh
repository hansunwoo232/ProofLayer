#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

write_local_env() {
  umask 077
  local generated_password
  generated_password="$(openssl rand -hex 24)"
  {
    printf 'SPLUNK_PASSWORD=%s\n' "${generated_password}"
    printf 'SPLUNK_GENERAL_TERMS=--accept-sgt-current-at-splunk-com\n'
  } > "${ENV_FILE}"
}

if ! command -v openssl >/dev/null 2>&1; then
  echo "OpenSSL is required to generate the local Splunk password." >&2
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  write_local_env
  echo "Created an ignored local environment file with a generated password."
else
  SPLUNK_PASSWORD=""
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  if (( ${#SPLUNK_PASSWORD} < 16 )); then
    invalid_env="${ENV_FILE}.invalid-$(date -u +%Y%m%dT%H%M%SZ)"
    mv "${ENV_FILE}" "${invalid_env}"
    write_local_env
    echo "Replaced an invalid local environment file; the previous file was preserved."
  else
    echo "Using the existing ignored local environment file."
  fi
fi

docker compose --project-directory "${SCRIPT_DIR}" --env-file "${ENV_FILE}" \
  -f "${SCRIPT_DIR}/compose.yaml" config >/dev/null

docker compose --project-directory "${SCRIPT_DIR}" --env-file "${ENV_FILE}" \
  -f "${SCRIPT_DIR}/compose.yaml" up -d

echo "Splunk startup requested. Run ./wait-until-ready.sh next."
