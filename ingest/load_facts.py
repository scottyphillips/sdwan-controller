#!/usr/bin/env python3
"""Read facts.json from playbook output and upsert into core.devices + core.device_facts."""
import json
import os
import sys
from pathlib import Path

import psycopg
from psycopg.rows import dict_row

FACTS_PATH = Path(os.environ.get("FACTS_PATH", "/out/facts.json"))


def main() -> None:
    if not FACTS_PATH.exists():
        print(f"Facts file not found: {FACTS_PATH}", file=sys.stderr)
        sys.exit(1)

    with open(FACTS_PATH) as f:
        payload = json.load(f)

    mgmt_ip = payload.get("mgmt_ip")
    hostname = payload.get("hostname") or mgmt_ip or "unknown"
    vendor = payload.get("vendor") or "juniper"
    platform = payload.get("platform") or "junos"
    facts = payload.get("facts") or {}

    conninfo = (
        f"host={os.environ.get('POSTGRES_HOST', '127.0.0.1')} "
        f"port={os.environ.get('POSTGRES_PORT', '5432')} "
        f"user={os.environ.get('POSTGRES_USER', 'admin')} "
        f"password={os.environ.get('POSTGRES_PASSWORD', '')} "
        f"dbname={os.environ.get('POSTGRES_DB', 'sdwan_core')}"
    )

    with psycopg.connect(conninfo, row_factory=dict_row) as conn:
        with conn.cursor() as cur:
            # Upsert device by hostname (unique)
            cur.execute(
                """
                INSERT INTO core.devices (hostname, vendor, platform, mgmt_ip, status)
                VALUES (%s, %s, %s, %s, 'online')
                ON CONFLICT (hostname)
                DO UPDATE SET
                    mgmt_ip = EXCLUDED.mgmt_ip,
                    vendor = EXCLUDED.vendor,
                    platform = EXCLUDED.platform,
                    status = 'online'
                RETURNING id
                """,
                (hostname, vendor, platform, mgmt_ip),
            )
            row = cur.fetchone()
            device_id = row["id"]

            # Insert facts snapshot
            cur.execute(
                """
                INSERT INTO core.device_facts (device_id, facts)
                VALUES (%s, %s::jsonb)
                """,
                (str(device_id), json.dumps(facts)),
            )

        conn.commit()

    print(f"Device: {hostname} ({mgmt_ip}) -> device_id={device_id}")
    print("Facts stored in core.device_facts.")


if __name__ == "__main__":
    main()
