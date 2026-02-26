# Ingest container: Ansible + Juniper collection, writes device facts to Postgres
FROM python:3.12-slim

WORKDIR /app

# ssh-keyscan for adding device host keys to known_hosts
RUN apt-get update && apt-get install -y --no-install-recommends openssh-client && rm -rf /var/lib/apt/lists/*

# Install Ansible and Python deps (build context = repo root from compose)
COPY ingest/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Juniper Junos collection (uses NETCONF; requires ncclient)
RUN ansible-galaxy collection install junipernetworks.junos --force

# Playbooks and loader
COPY ingest/playbooks /playbooks
COPY ingest/load_facts.py .
COPY ingest/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Output dir for facts JSON (playbook writes here)
RUN mkdir -p /out

ENTRYPOINT ["/entrypoint.sh"]
