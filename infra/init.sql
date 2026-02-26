-- Create a schema to keep things organized
CREATE SCHEMA IF NOT EXISTS core;

-- 1. Devices Table (The "What")
CREATE TABLE core.devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname TEXT NOT NULL UNIQUE,
    vendor TEXT NOT NULL, -- e.g., 'cisco', 'juniper', 'linux'
    platform TEXT,        -- e.g., 'ios-xe', 'junos', 'ubuntu'
    mgmt_ip INET NOT NULL,
    status TEXT DEFAULT 'offline',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. IPAM: Subnets (The "Where")
CREATE TABLE core.subnets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network CIDR NOT NULL UNIQUE, -- e.g., '192.168.10.0/24'
    description TEXT,
    vlan_id INTEGER,
    site_name TEXT DEFAULT 'Whittlesea'
);

-- 3. IPAM: IP Addresses (The "Who")
CREATE TABLE core.ip_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    address INET NOT NULL UNIQUE,
    subnet_id UUID REFERENCES core.subnets(id),
    status TEXT DEFAULT 'allocated', -- 'allocated', 'reserved', 'discovered'
    mac_address MACADDR,
    last_seen TIMESTAMPTZ
);

-- 4. Interfaces (The "How")
CREATE TABLE core.interfaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES core.devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT, -- 'wan', 'lan', 'tunnel'
    is_up BOOLEAN DEFAULT false
);

-- 5. Device Audits (LLM discovery / cognitive plane)
CREATE TABLE core.device_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL,
    tier TEXT NOT NULL,
    hostname TEXT NOT NULL,
    confidence INTEGER NOT NULL,
    logic TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. Device Facts (structured ingest from Ansible/Netmiko)
CREATE TABLE core.device_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES core.devices(id) ON DELETE CASCADE,
    gathered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    facts JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_device_facts_device_id ON core.device_facts(device_id);
CREATE INDEX IF NOT EXISTS idx_device_facts_gathered_at ON core.device_facts(gathered_at DESC);