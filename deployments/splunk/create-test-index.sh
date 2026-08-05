#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
CONTAINER_NAME="prooflayer-splunk"
INDEX_NAME="prooflayer_test"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}. Run ./setup.sh first." >&2
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

if docker exec --user splunk "${CONTAINER_NAME}" /opt/splunk/bin/splunk list index "${INDEX_NAME}" \
  -auth "admin:${SPLUNK_PASSWORD}" >/dev/null 2>&1; then
  echo "Index ${INDEX_NAME} already exists."
else
  docker exec --user splunk "${CONTAINER_NAME}" /opt/splunk/bin/splunk add index "${INDEX_NAME}" \
    -datatype event \
    -maxTotalDataSizeMB 512 \
    -auth "admin:${SPLUNK_PASSWORD}" >/dev/null
  echo "Created index ${INDEX_NAME}."
fi

docker exec --user splunk "${CONTAINER_NAME}" /opt/splunk/bin/splunk list index "${INDEX_NAME}" \
  -auth "admin:${SPLUNK_PASSWORD}" >/dev/null

echo "Verified index ${INDEX_NAME}."
