package ru.warmhouse.devices;

import java.time.Instant;
import java.util.List;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;

@Repository
public class DeviceRepository {
    public record Device(int id, String name, String type, String location, double value,
                         String unit, String status, Instant lastUpdated, Instant createdAt) {}

    private static final RowMapper<Device> MAPPER = (row, index) -> new Device(
        row.getInt("id"), row.getString("name"), row.getString("type"), row.getString("location"),
        row.getDouble("value"), row.getString("unit"), row.getString("status"),
        row.getTimestamp("last_updated").toInstant(), row.getTimestamp("created_at").toInstant());

    private final JdbcTemplate jdbc;

    public DeviceRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public void checkConnection() {
        jdbc.queryForObject("SELECT 1", Integer.class);
    }

    public List<Device> findAll() {
        return jdbc.query("SELECT * FROM sensors ORDER BY id", MAPPER);
    }

    public Device findById(int id) {
        return jdbc.queryForObject("SELECT * FROM sensors WHERE id = ?", MAPPER, id);
    }

    public Device create(DeviceController.CreateDevice request) {
        return jdbc.queryForObject("""
            INSERT INTO sensors (name, type, location, unit)
            VALUES (?, ?, ?, ?) RETURNING *
            """, MAPPER, request.name(), request.type(), request.location(),
            request.unit() == null ? "" : request.unit());
    }

    public Device update(int id, DeviceController.UpdateDevice request) {
        return jdbc.queryForObject("""
            UPDATE sensors SET
                name = COALESCE(NULLIF(?, ''), name),
                type = COALESCE(NULLIF(?, ''), type),
                location = COALESCE(NULLIF(?, ''), location),
                value = COALESCE(?, value),
                unit = COALESCE(NULLIF(?, ''), unit),
                status = COALESCE(NULLIF(?, ''), status),
                last_updated = NOW()
            WHERE id = ? RETURNING *
            """, MAPPER, request.name(), request.type(), request.location(), request.value(),
            request.unit(), request.status(), id);
    }

    public boolean delete(int id) {
        return jdbc.update("DELETE FROM sensors WHERE id = ?", id) > 0;
    }
}
