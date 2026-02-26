#!/bin/bash

# Configuration
CONTAINER_NAME="sdwan_llm"
MODEL_NAME="deepseek-r1:14b"
COMPOSE_FILE="./infra/docker-compose.yml"

echo "🚀 Starting SD-WAN Stack (Postgres, Redis, Ollama)..."
# Using 'docker compose' (V2) as we confirmed this works on your WSL2 setup
docker compose --env-file .env -f $COMPOSE_FILE up -d

echo "⏳ Waiting for Postgres..."
until docker exec sdwan_db pg_isready -U admin -d sdwan_core > /dev/null 2>&1; do
  sleep 2
done
echo "✅ Postgres is ready."
echo "📋 Ensuring core schema and device_audits table..."
# Idempotent: safe for existing DBs that were created before this table was in init.sql
source .env 2>/dev/null || true
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" sdwan_db psql -U "${POSTGRES_USER:-admin}" -d "${POSTGRES_DB:-sdwan_core}" -c "
CREATE SCHEMA IF NOT EXISTS core;
CREATE TABLE IF NOT EXISTS core.device_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL,
    tier TEXT NOT NULL,
    hostname TEXT NOT NULL,
    confidence INTEGER NOT NULL,
    logic TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS core.device_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES core.devices(id) ON DELETE CASCADE,
    gathered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    facts JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_device_facts_device_id ON core.device_facts(device_id);
CREATE INDEX IF NOT EXISTS idx_device_facts_gathered_at ON core.device_facts(gathered_at DESC);
" 2>/dev/null && echo "✅ Database schema ready." || echo "⚠️  Migration skipped (Postgres not ready or .env missing)."

echo "⏳ Waiting for Ollama engine to wake up..."
# We check from the WSL2 side
until curl -s http://localhost:11434/api/tags > /dev/null; do
  echo "...still warming up (checking API from WSL2)..."
  sleep 3
done

echo "✅ AI Engine is responsive!"
# CHECK IF MODEL ALREADY EXISTS
echo "🔍 Checking for $MODEL_NAME..."
if docker exec $CONTAINER_NAME ollama list | grep -q "$MODEL_NAME"; then
    echo "✅ Model already present. Skipping download."
else
    echo "📥 Model not found. Pulling $MODEL_NAME (9GB) to Docker Volume..."
    docker exec -it $CONTAINER_NAME ollama pull $MODEL_NAME
fi

echo "⚙️  Verifying RTX 5070 Ti visibility..."
# Ensure the container can see the GPU before starting the heavy download
if docker exec $CONTAINER_NAME nvidia-smi | grep -q "5070 Ti"; then
    echo "✅ GPU detected! Commencing model pull."
else
    echo "⚠️  Warning: GPU not detected in container. Pulling model anyway, but it may run on CPU."
fi

echo "📥 Pulling $MODEL_NAME (9GB)..."
# This will show a live progress bar
docker exec -it $CONTAINER_NAME ollama pull $MODEL_NAME

echo "------------------------------------------------"
echo "✅ STACK IS LIVE"
echo "Postgres: Port 5432"
echo "Redis: Port 6379"
echo "Ollama: Port 11434"
echo "------------------------------------------------"
docker ps