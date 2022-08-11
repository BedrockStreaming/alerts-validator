# alerts-validator

This project aims to define if currently used alerting rules in Victoria Metrics are valid.

By valid, we mean that alerting rules should use metrics that are still available, and fire if needed.

A metric could disappear or be renamed after upgrading an infracstructre's component. Alerts-validator help us to have up-to-date alerting rules.

This project has a continuous loop which get alerting rules from VM Alert endpoint, then analyze them with VM Select endpoint, and expose them as Prometheus Metric.

## Example

Let's say you are using this alerting rule (from [Awesome Prometheus Alerts](https://awesome-prometheus-alerts.grep.to/rules.html#rule-host-and-hardware-1-1)) : 

```
- alert: HostOutOfMemory
  expr: node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100 < 10
```

Let's say `node_memory_MemAvailable_bytes` as been renamed because of an upgrade of Prometheus Node Exporter last week.

Alerts-validator will analyze every metric's previous rules by querying VM Select endpoint :

```
present_over_time(node_memory_MemAvailable_bytes[167h]) offset 1h   --> No datapoints
present_over_time(node_memory_MemTotal_bytes[167h]) offset 1h       --> Some datapoints
```

If one of those metrics doesn't have any datapoints, the alerting rule `HostOutOfMemory` will be define as invalid for time range [1h-168h].

You can now be alerted that `HostOutOfMemory` alerting rule need to be updated.

## Build

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o alerts-validator-linux-amd64
```

## Compatibility

Works from Victoria Metrics v1.76.0 to last

## Configuration

### Command line

```sh
Usage of alerts-validator:
  -conf string
        Config path (default "config.yaml")
```

### YAML

```yaml
listenAddress: 0.0.0.0      # Listen address for metrics endpoint
listenPort: 9345            # Listen port for metrics endpoint
computeInterval: 168h       # Interval between 2 computes (Valid time units are "s", "m", "h". )
validityCheckIntervals:     # Check if there are datapoints betwen 2 intervals (Valid time units are "s", "m", "h". )
- 1h
- 168h
labelKeys:
- tenant
servers:
  - labelValues:                                                   # Added in metric label
    - my_tenant
    alertUrl: https://vmalert.cluster.local.                       # Endpoint to get alerts
    selectUrl: https://vm.cluster.local./select/000/prometheus     # Endpoint to validate metrics
```

## Metrics

With config :

```yaml
validityCheckIntervals:
- 1h
- 168h
- 672h
```

```
alertsvalidator_validity_range{range_from="1h",range_to="168h",alertname="my_alert",status="valid"}     0
alertsvalidator_validity_range{range_from="1h",range_to="168h",alertname="my_alert",status="invalid"}   1 # Invalid


alertsvalidator_validity_range{range_from="168h",range_to="672h",alertname="my_alert",status="valid"}   1 # Valid
alertsvalidator_validity_range{range_from="168h",range_to="672h",alertname="my_alert",status="invalid"} 0
```

## Logs

Some logs are available, they display more informations on alert-validators previous metrics.

```json
{
    "level": "info",
    "alertname": "myInvalidAlert",
    "is_vector_present": {
        "1h-168h": {
            "up{instance=\"my_server\"}": true,
        },
        "168h-672h": {
            "up{instance=\"my_server\"}": false,
        },
    },
    "is_valid": {
        "1h-168h": true,
        "168h-672h": false,
    },
    "time": ---
}
```

## Contributors

- [Arthur Zinck](https://github.com/arthurzinck)
- [Tanguy Falconnet](https://github.com/tanguyfalconnet)
