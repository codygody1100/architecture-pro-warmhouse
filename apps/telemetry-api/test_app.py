import tempfile
import unittest
from pathlib import Path

from app import create_app


class TelemetryTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = str(Path(self.directory.name) / "telemetry.db")
        self.client = create_app(self.path).test_client()
        self.reading = {
            "device_id": 1, "metric": "temperature", "value": 0, "unit": "°C",
            "recorded_at": "2026-11-14T12:00:00Z",
        }

    def test_history_is_persistent_filtered_and_sorted_by_measurement_time(self):
        for device_id, timestamp in [(1, "13:00:00+03:00"), (1, "12:00:00Z"), (2, "14:00:00Z")]:
            response = self.client.post("/readings", json={
                **self.reading, "device_id": device_id, "recorded_at": f"2026-11-14T{timestamp}",
            })
            self.assertEqual(response.status_code, 201)
        restarted = create_app(self.path).test_client()
        history = restarted.get("/readings?device_id=1").get_json()
        self.assertEqual(len(history), 2)
        self.assertEqual(history[0]["recorded_at"], "2026-11-14T12:00:00.000000+00:00")
        self.assertEqual(history[1]["recorded_at"], "2026-11-14T10:00:00.000000+00:00")
        self.assertEqual(restarted.get("/readings?device_id=1&limit=1").get_json(), history[:1])
        self.assertEqual(restarted.get("/readings?device_id=99").get_json(), [])

    def test_invalid_readings_and_query(self):
        for changes in [{"device_id": 0}, {"device_id": True}, {"device_id": 2**63},
                        {"value": True}, {"value": "21"}, {"value": float("inf")},
                        {"metric": ""}, {"recorded_at": "2026-11-14T12:00:00"}]:
            with self.subTest(changes=changes):
                self.assertEqual(self.client.post("/readings", json={**self.reading, **changes}).status_code, 400)
        for query in ["", "device_id=x", "device_id=0", "device_id=1&limit=1001", "device_id=1&limit=0"]:
            self.assertEqual(self.client.get("/readings?" + query).status_code, 400)
        self.assertEqual(self.client.post("/readings", json=[]).status_code, 400)
        self.assertEqual(self.client.get("/readings?device_id=1").get_json(), [])


if __name__ == "__main__":
    unittest.main()
