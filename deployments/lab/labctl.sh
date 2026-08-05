#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "$script_dir/../.." && pwd)"
workspace_root="$(cd "$project_root/../.." && pwd)"

if [[ -f "$script_dir/lab.env" ]]; then
  # shellcheck disable=SC1091
  source "$script_dir/lab.env"
fi

state_dir="${PROOFLAYER_LAB_STATE_DIR:-$workspace_root/work/prooflayer-lab}"
memory_mb="${LAB_MEMORY_MB:-8192}"
cpu_count="${LAB_CPUS:-4}"
disk_size="${LAB_DISK_SIZE:-80G}"
network_mode="${LAB_NETWORK_MODE:-isolated}"
windows_iso="${WINDOWS_ISO:-}"
tools_iso="${LAB_TOOLS_ISO_OVERRIDE:-${LAB_TOOLS_ISO:-$state_dir/prooflayer-tools.iso}}"

qemu_bin="${QEMU_BIN:-$(command -v qemu-system-aarch64 || true)}"
qemu_img="${QEMU_IMG:-$(command -v qemu-img || true)}"
qemu_share="${QEMU_SHARE:-/opt/homebrew/opt/qemu/share/qemu}"
uefi_code="${UEFI_CODE:-$qemu_share/edk2-aarch64-code.fd}"
uefi_vars_template="${UEFI_VARS_TEMPLATE:-$qemu_share/edk2-arm-vars.fd}"

disk_path="$state_dir/windows-11-arm64.qcow2"
uefi_vars="$state_dir/windows-11-arm64-vars.fd"
tpm_dir="$state_dir/tpm"
tpm_socket="/tmp/prooflayer-swtpm-${UID}.sock"
tpm_log="$state_dir/swtpm.log"
checkpoint_root="$state_dir/checkpoints"

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_file() {
  [[ -f "$1" ]] || die "Required file not found: $1"
}

preflight() {
  [[ -n "$qemu_bin" ]] || die "qemu-system-aarch64 is not installed."
  [[ -n "$qemu_img" ]] || die "qemu-img is not installed."
  require_file "$uefi_code"
  require_file "$uefi_vars_template"

  if [[ "$network_mode" == "siem" && ! -x /usr/bin/nc ]]; then
    die "SIEM mode requires the host relay executable: /usr/bin/nc"
  fi

  printf 'QEMU: %s\n' "$qemu_bin"
  printf 'QEMU image tool: %s\n' "$qemu_img"
  printf 'UEFI code: %s\n' "$uefi_code"
  printf 'UEFI variables template: %s\n' "$uefi_vars_template"
  printf 'State directory: %s\n' "$state_dir"
  printf 'Network mode: %s\n' "$network_mode"

  if command -v swtpm >/dev/null 2>&1; then
    printf 'Virtual TPM: available\n'
  else
    printf 'Virtual TPM: missing (install swtpm before first boot)\n'
  fi

  if [[ -n "$windows_iso" && -f "$windows_iso" ]]; then
    printf 'Windows ISO: %s\n' "$windows_iso"
  else
    printf 'Windows ISO: not configured\n'
  fi
}

prepare() {
  preflight
  mkdir -p "$state_dir" "$tpm_dir"

  if [[ ! -f "$disk_path" ]]; then
    "$qemu_img" create -f qcow2 "$disk_path" "$disk_size"
  else
    printf 'Disk already exists: %s\n' "$disk_path"
  fi

  if [[ ! -f "$uefi_vars" ]]; then
    cp "$uefi_vars_template" "$uefi_vars"
  else
    printf 'UEFI state already exists: %s\n' "$uefi_vars"
  fi

  printf 'Lab state prepared successfully.\n'
}

create_checkpoint() {
  local checkpoint_name="${1:-}"
  [[ "$checkpoint_name" =~ ^[a-z0-9][a-z0-9._-]*$ ]] ||
    die "Checkpoint name must use lowercase letters, numbers, dots, dashes, or underscores."

  require_file "$disk_path"
  require_file "$uefi_vars"

  local checkpoint_dir="$checkpoint_root/$checkpoint_name"
  [[ ! -e "$checkpoint_dir" ]] || die "Checkpoint already exists: $checkpoint_dir"

  "$qemu_img" info "$disk_path" >/dev/null
  mkdir -p "$checkpoint_dir"
  cp "$uefi_vars" "$checkpoint_dir/uefi-vars.fd"
  if [[ -d "$tpm_dir" ]]; then
    cp -R "$tpm_dir" "$checkpoint_dir/tpm"
  fi

  "$qemu_img" snapshot -c "$checkpoint_name" "$disk_path"
  printf 'Checkpoint created: %s\n' "$checkpoint_name"
  printf 'Firmware and TPM state: %s\n' "$checkpoint_dir"
}

list_checkpoints() {
  require_file "$disk_path"
  "$qemu_img" snapshot -l "$disk_path"
}

run_vm() {
  local include_installer="$1"
  local include_tools="$2"
  require_file "$disk_path"
  require_file "$uefi_vars"
  command -v swtpm >/dev/null 2>&1 ||
    die "swtpm is required for a Windows 11 TPM 2.0 device."

  if [[ "$include_installer" == "yes" ]]; then
    [[ -n "$windows_iso" ]] || die "Set WINDOWS_ISO in lab.env."
    require_file "$windows_iso"
  fi
  if [[ "$include_tools" == "yes" ]]; then
    require_file "$tools_iso"
  fi

  mkdir -p "$tpm_dir"
  rm -f "$tpm_socket"

  swtpm socket \
    --tpm2 \
    --tpmstate "dir=$tpm_dir" \
    --ctrl "type=unixio,path=$tpm_socket" \
    --log "file=$tpm_log,level=20" &
  local swtpm_pid=$!
  trap 'kill "$swtpm_pid" 2>/dev/null || true' EXIT

  for _ in {1..50}; do
    [[ -S "$tpm_socket" ]] && break
    sleep 0.1
  done
  [[ -S "$tpm_socket" ]] || die "Virtual TPM socket did not become ready."

  local net_args=()
  case "$network_mode" in
    offline)
      net_args=(-nic none)
      ;;
    isolated)
      net_args=(-nic user,model=virtio-net-pci,restrict=on)
      ;;
    siem)
      net_args=(-nic "user,model=virtio-net-pci,restrict=on,guestfwd=tcp:10.0.2.100:8088-cmd:/usr/bin/nc 127.0.0.1 8088")
      ;;
    nat)
      [[ "${ALLOW_INTERNET:-NO}" == "YES" ]] ||
        die "NAT mode requires ALLOW_INTERNET=YES in lab.env."
      net_args=(-nic user,model=virtio-net-pci)
      ;;
    *)
      die "Unknown LAB_NETWORK_MODE: $network_mode"
      ;;
  esac

  local args=(
    -name "ProofLayer Windows Lab"
    -machine virt,accel=hvf,highmem=on
    -cpu host
    -smp "$cpu_count"
    -m "$memory_mb"
    -drive "if=pflash,format=raw,readonly=on,file=$uefi_code"
    -drive "if=pflash,format=raw,file=$uefi_vars"
    -drive "if=none,format=qcow2,file=$disk_path,id=system"
    -device "nvme,drive=system,serial=PLLAB001,bootindex=2"
    -device qemu-xhci
    -device usb-kbd
    -device usb-tablet
    -device ramfb
    -display cocoa
    -chardev "socket,id=chrtpm,path=$tpm_socket"
    -tpmdev "emulator,id=tpm0,chardev=chrtpm"
    -device "tpm-tis-device,tpmdev=tpm0"
    "${net_args[@]}"
  )

  if [[ "$include_installer" == "yes" ]]; then
    args+=(
      -drive "if=none,media=cdrom,readonly=on,file=$windows_iso,id=installmedia"
      -device "usb-storage,drive=installmedia,removable=true,bootindex=1"
      -boot menu=on
    )
  fi

  if [[ "$include_tools" == "yes" ]]; then
    args+=(
      -drive "if=none,media=cdrom,readonly=on,file=$tools_iso,id=toolsmedia"
      -device "usb-storage,drive=toolsmedia,removable=true"
    )
  fi

  "$qemu_bin" "${args[@]}"

  kill "$swtpm_pid" 2>/dev/null || true
  wait "$swtpm_pid" 2>/dev/null || true
  trap - EXIT
}

case "${1:-}" in
  preflight)
    preflight
    ;;
  prepare)
    prepare
    ;;
  install)
    run_vm yes no
    ;;
  start)
    run_vm no no
    ;;
  tools)
    run_vm no yes
    ;;
  checkpoint)
    create_checkpoint "${2:-}"
    ;;
  checkpoints)
    list_checkpoints
    ;;
  *)
    printf 'Usage: %s {preflight|prepare|install|start|tools|checkpoint NAME|checkpoints}\n' "$0"
    exit 2
    ;;
esac
