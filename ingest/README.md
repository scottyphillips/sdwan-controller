# Network Ingest (Ansible + Juniper)

This container runs Ansible playbooks to gather structured facts from network devices (e.g. Juniper vSRX) and load them into Postgres (`core.devices`, `core.device_facts`).

## vSRX setup (NETCONF)

The Juniper playbook uses **NETCONF** (port 830). On your vSRX:

1. Enable NETCONF over SSH:
   ```junos
   set system services netconf ssh port 830
   commit
   ```
2. Ensure SSH is enabled and you can log in with the user/password you will set in `.env`.

## Host keys (known_hosts)

Before running the playbook, the entrypoint runs `ssh-keyscan` against `JUNIPER_HOST` on the NETCONF port and appends the key to `/root/.ssh/known_hosts` inside the container. That directory is persisted in a Docker volume (`ingest_known_hosts`), so:

- **First run:** The device key is fetched and stored; later runs reuse it and host key verification stays on.
- **New device:** Add the new host to `.env` and run ingest again; its key will be scanned and added.

To pre-populate from your host (e.g. copy an existing `~/.ssh/known_hosts`):

```bash
docker run --rm -v sdwan-controller_ingest_known_hosts:/data -v "$HOME/.ssh/known_hosts:/src:ro" alpine sh -c "cp -r /data/. /tmp/ 2>/dev/null; mkdir -p /tmp/.ssh; cat /src >> /tmp/.ssh/known_hosts 2>/dev/null; cp -r /tmp/. /data/"
```

(Replace `sdwan-controller_ingest_known_hosts` with your project’s volume name if different.)

## Environment variables

Add to your repo root `.env` (or pass when running):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JUNIPER_HOST` | Yes | — | vSRX management IP (reachable from host) |
| `JUNIPER_USER` | No | `admin` | SSH/NETCONF user |
| `JUNIPER_PASSWORD` | Yes | — | SSH/NETCONF password |
| `JUNIPER_NETCONF_PORT` | No | `830` | NETCONF port |
| `POSTGRES_*` | — | — | Same as main stack (used when loading facts) |

## Run ingest

From repo root, with the main stack already up (so Postgres is on 5432):

```bash
docker compose -f infra/docker-compose.yml --env-file .env --profile ingest run --rm ingest
```

The container uses `network_mode: host` so it can reach both the vSRX (on your LAN/VMware network) and Postgres on `127.0.0.1:5432`.

## What gets stored

- **core.devices**: one row per device (hostname, vendor=juniper, platform=junos, mgmt_ip, status=online). Upserted by hostname.
- **core.device_facts**: one row per run with full facts JSON (version, hardware, interfaces, etc.).

Query after a run:

```sql
SELECT id, hostname, vendor, mgmt_ip FROM core.devices;
SELECT device_id, gathered_at, facts->'ansible_net_version' AS version FROM core.device_facts ORDER BY gathered_at DESC LIMIT 1;
```
