#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="prooflayer-splunk"
MAX_ATTEMPTS=60

for ((attempt = 1; attempt <= MAX_ATTEMPTS; attempt++)); do
  status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${CONTAINER_NAME}" 2>/dev/null || true)"

  if [[ "${status}" == "healthy" ]]; then
    if docker logs "${CONTAINER_NAME}" 2>&1 | grep -q "Ansible playbook complete"; then
      echo "Splunk is healthy and initial provisioning is complete."
      exit 0
    fi
  fi

  if [[ "${status}" == "exited" || "${status}" == "dead" ]]; then
    echo "Splunk stopped before becoming healthy." >&2
    docker logs --tail 80 "${CONTAINER_NAME}" >&2
    exit 1
  fi

  sleep 5
done

echo "Splunk did not become healthy within the expected window." >&2
docker logs --tail 80 "${CONTAINER_NAME}" >&2
exit 1
