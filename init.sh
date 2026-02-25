#!/bin/bash

# Configuration
CONTAINER_NAME="sdwan_llm"
MODEL_NAME="deepseek-r1:14b"
COMPOSE_FILE="./infra/docker-compose.yml"

echo "🚀 Starting SD-WAN Stack (Postgres, Redis, Ollama)..."
# Using 'docker compose' (V2) as we confirmed this works on your WSL2 setup
docker compose -f $COMPOSE_FILE up -d

echo "⏳ Waiting for Ollama engine to wake up..."
# We check from the WSL2 side
until curl -s http://localhost:11434/api/tags > /dev/null; do
  echo "...still warming up (checking API from WSL2)..."
  sleep 3
done

echo "✅ AI Engine is responsive!"

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