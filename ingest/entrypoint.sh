#!/usr/bin/env bash
set -e

JUNIPER_HOST="${JUNIPER_HOST:?Set JUNIPER_HOST}"
NETCONF_PORT="${JUNIPER_NETCONF_PORT:-830}"

# Ensure known_hosts exists and add the device key (persists via volume mount at /root/.ssh)
mkdir -p /root/.ssh
touch /root/.ssh/known_hosts
if ! grep -q "\[${JUNIPER_HOST}\]:${NETCONF_PORT}" /root/.ssh/known_hosts 2>/dev/null; then
  echo "Adding host key for ${JUNIPER_HOST}:${NETCONF_PORT} to known_hosts..."
  ssh-keyscan -p "$NETCONF_PORT" "$JUNIPER_HOST" >> /root/.ssh/known_hosts 2>/dev/null || {
    echo "Warning: ssh-keyscan failed (device down or unreachable). Will try playbook anyway."
  }
fi

# Generate inventory from env (NETCONF port 830)
INVENTORY="/tmp/inventory.yml"
cat > "$INVENTORY" << EOF
---
all:
  children:
    juniper:
      hosts:
        vsrx:
          ansible_host: "${JUNIPER_HOST}"
          ansible_user: "${JUNIPER_USER:-admin}"
          ansible_password: "${JUNIPER_PASSWORD:?Set JUNIPER_PASSWORD}"
          ansible_connection: netconf
          ansible_network_os: junipernetworks.junos.junos
          ansible_port: "${NETCONF_PORT}"
          ansible_ssh_common_args: '-o UserKnownHostsFile=/root/.ssh/known_hosts'
EOF

echo "Target: $JUNIPER_HOST (NETCONF port $NETCONF_PORT)"

# Run playbook (host key verification via known_hosts above)
ansible-playbook -i "$INVENTORY" /playbooks/juniper-facts.yml -v

# Load into Postgres (when using host network, Postgres is at POSTGRES_HOST)
python3 /app/load_facts.py
