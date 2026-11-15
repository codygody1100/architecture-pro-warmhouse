package ru.warmhouse.devices;

import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import java.util.List;
import java.util.Map;
import org.springframework.dao.DataAccessException;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;

@RestController
public class DeviceController {
    public record CreateDevice(@NotBlank @Size(max = 100) String name,
                               @NotBlank @Size(max = 50) String type,
                               @NotBlank @Size(max = 100) String location,
                               @Size(max = 20) String unit) {}

    public record UpdateDevice(@Size(max = 100) String name, @Size(max = 50) String type,
                               @Size(max = 100) String location, Double value,
                               @Size(max = 20) String unit, @Size(max = 20) String status) {}

    public record UpdateValue(@NotNull Double value, @NotBlank @Size(max = 20) String status) {}

    private final DeviceRepository repository;

    public DeviceController(DeviceRepository repository) {
        this.repository = repository;
    }

    @GetMapping("/health")
    public Map<String, String> health() {
        repository.checkConnection();
        return Map.of("status", "ok");
    }

    @GetMapping("/devices")
    public List<DeviceRepository.Device> list() {
        return repository.findAll();
    }

    @GetMapping("/devices/{id}")
    public DeviceRepository.Device get(@PathVariable int id) {
        return repository.findById(id);
    }

    @PostMapping("/devices")
    @ResponseStatus(HttpStatus.CREATED)
    public DeviceRepository.Device create(@Valid @RequestBody CreateDevice request) {
        return repository.create(request);
    }

    @PutMapping("/devices/{id}")
    public DeviceRepository.Device update(@PathVariable int id, @Valid @RequestBody UpdateDevice request) {
        checkValue(request.value());
        return repository.update(id, request);
    }

    @PatchMapping("/devices/{id}/value")
    public Map<String, String> updateValue(@PathVariable int id, @Valid @RequestBody UpdateValue request) {
        checkValue(request.value());
        repository.update(id, new UpdateDevice(null, null, null, request.value(), null, request.status()));
        return Map.of("message", "Sensor value updated successfully");
    }

    @DeleteMapping("/devices/{id}")
    public Map<String, String> delete(@PathVariable int id) {
        if (!repository.delete(id)) {
            throw new ResponseStatusException(HttpStatus.NOT_FOUND, "Device not found");
        }
        return Map.of("message", "Sensor deleted successfully");
    }

    private void checkValue(Double value) {
        if (value != null && !Double.isFinite(value)) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "Value must be finite");
        }
    }

    @ExceptionHandler(EmptyResultDataAccessException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    public Map<String, String> notFound() {
        return Map.of("error", "Device not found");
    }

    @ExceptionHandler(DataAccessException.class)
    @ResponseStatus(HttpStatus.SERVICE_UNAVAILABLE)
    public Map<String, String> databaseError() {
        return Map.of("error", "Device database is unavailable");
    }
}
