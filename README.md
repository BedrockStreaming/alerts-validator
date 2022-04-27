# alerts-validator

Define which Prometheus alert has no DataPoint over different time range: 1 day, 30 days, and 90 days.

Continuous loop wich get alerts from Alert endpoint, then analyze them with Select endpoint.

## Build

```sh
GOOS=linux GOARCH=amd64 go build -o alerts-validator-linux-amd64
```

## Compatibility

Works with Victoria Metrics v1.76.0

## Metrics

```
alertsvalidator_present_over_1day
alertsvalidator_present_over_30days
alertsvalidator_present_over_90days
```

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
interval: 10                # Interval between 2 compute in minutes

servers:
  - tenant: "my_tenant"                                             # Added in metric label
    alertUrl: https://vmalert.cluster.local.                        # Endpoint to get alerts
    selectUrl: https://vm.cluster.local./select/000/prometheus      # Endpoint to validate metrics
```

## Contributors

- [Arthur Zinck](https://github.com/arthurzinck)
- [Tanguy Falconnet](https://github.com/tanguyfalconnet)
