#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

docker compose up --build -d --wait
echo "Smart Home API: http://localhost:8080"
echo "Temperature API: http://localhost:8081"
echo "Device API: http://localhost:8083"
echo "Telemetry API: http://localhost:8084"
