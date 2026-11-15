import math
import os
import sqlite3
from contextlib import contextmanager
from datetime import datetime, timezone
from pathlib import Path

from flask import Flask, jsonify, request


class TelemetryRepository:
    def __init__(self, path):
        self.path = path
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        with self.connect() as connection:
            connection.executescript("""
                CREATE TABLE IF NOT EXISTS readings (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    device_id INTEGER NOT NULL,
                    metric TEXT NOT NULL,
                    value REAL NOT NULL,
                    unit TEXT NOT NULL,
                    recorded_at TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS readings_device_time
                ON readings (device_id, recorded_at DESC, id DESC);
            """)

    @contextmanager
    def connect(self):
        connection = sqlite3.connect(self.path, timeout=5)
        connection.row_factory = sqlite3.Row
        try:
            with connection:
                yield connection
        finally:
            connection.close()

    def save(self, reading):
        with self.connect() as connection:
            cursor = connection.execute("""
                INSERT INTO readings (device_id, metric, value, unit, recorded_at)
                VALUES (:device_id, :metric, :value, :unit, :recorded_at)
            """, reading)
            return {"id": cursor.lastrowid, **reading}

    def history(self, device_id, limit):
        with self.connect() as connection:
            rows = connection.execute("""
                SELECT * FROM readings WHERE device_id = ?
                ORDER BY recorded_at DESC, id DESC LIMIT ?
            """, (device_id, limit))
            return [dict(row) for row in rows]

    def check_connection(self):
        with self.connect() as connection:
            connection.execute("SELECT 1")


def create_app(database_path=None):
    app = Flask(__name__)
    app.config["MAX_CONTENT_LENGTH"] = 1024 * 1024
    repository = TelemetryRepository(database_path or os.getenv("DATABASE_PATH", "telemetry.db"))

    @app.get("/health")
    def health():
        repository.check_connection()
        return {"status": "ok"}

    @app.post("/readings")
    def save_reading():
        data = request.get_json(silent=True)
        if not isinstance(data, dict):
            return {"error": "Expected a JSON object"}, 400
        device_id = data.get("device_id")
        value = data.get("value")
        metric = data.get("metric")
        unit = data.get("unit")
        if type(device_id) is not int or not 1 <= device_id <= 2147483647:
            return {"error": "device_id must be a positive integer"}, 400
        try:
            valid_value = type(value) in (int, float) and math.isfinite(value)
        except OverflowError:
            valid_value = False
        if not valid_value:
            return {"error": "value must be a finite number"}, 400
        if not isinstance(metric, str) or not metric.strip() or len(metric) > 50:
            return {"error": "metric must contain 1 to 50 characters"}, 400
        if not isinstance(unit, str) or len(unit) > 20:
            return {"error": "unit must be a string of up to 20 characters"}, 400
        try:
            timestamp = datetime.fromisoformat(data["recorded_at"].replace("Z", "+00:00"))
            if timestamp.tzinfo is None:
                raise ValueError("Timezone is required")
        except (KeyError, TypeError, AttributeError, ValueError):
            return {"error": "recorded_at must be an ISO 8601 timestamp with timezone"}, 400
        reading = {
            "device_id": device_id, "metric": metric, "value": value, "unit": unit,
            "recorded_at": timestamp.astimezone(timezone.utc).isoformat(timespec="microseconds"),
        }
        return jsonify(repository.save(reading)), 201

    @app.get("/readings")
    def history():
        try:
            device_id = int(request.args.get("device_id", ""))
            limit = int(request.args.get("limit", "100"))
            if not 1 <= device_id <= 2147483647 or not 1 <= limit <= 1000:
                raise ValueError()
        except ValueError:
            return {"error": "Provide a positive device_id and limit from 1 to 1000"}, 400
        return jsonify(repository.history(device_id, limit))

    @app.errorhandler(sqlite3.Error)
    def database_error(error):
        app.logger.error("Telemetry database error: %s", error)
        return {"error": "Telemetry database is unavailable"}, 503

    return app
