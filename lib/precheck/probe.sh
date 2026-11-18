#!/bin/bash
# vsno hardware probe. Runs on the node (chroot /host). Prints KEY=VALUE lines that
# lib/precheck parses. Read-only: no writes to the node.
echo "PRODUCT=$(cat /sys/class/dmi/id/product_name 2>/dev/null | tr -d '\n')"
echo "VIRT=$(systemd-detect-virt 2>/dev/null || true)"
if [ -c /dev/kvm ]; then echo "KVM=yes"; else echo "KVM=no"; fi
echo "NPROC=$(nproc)"
echo "MEM_TOTAL_KB=$(awk '/^MemTotal:/{print $2}' /proc/meminfo)"
echo "MEM_AVAILABLE_KB=$(awk '/^MemAvailable:/{print $2}' /proc/meminfo)"
echo "LSBLK=$(lsblk -J -b -o NAME,SIZE,TYPE,FSTYPE,PTTYPE,MOUNTPOINT,ROTA,RM 2>/dev/null | tr -d '\n')"
for l in /dev/disk/by-id/*; do
  [ -L "$l" ] || continue
  echo "BYID=$(basename "$(readlink -f "$l")") $(basename "$l")"
done
