# Nexmo Exporter for Prometheus
Fork from https://github.com/pzinovkin/nexmo_exporter optimized for deployment on Kubernetes.

[![Go Report Card](https://goreportcard.com/badge/github.com/g30rg3-stoica/nexmo_exporter)](https://goreportcard.com/report/github.com/g30rg3-stoica/nexmo_exporter)

Simple server that scraps Nexmo balance and exports it as Prometheus metrics.

### Changelog
- exposed metrics HTTP endpoint URL and port as program arguments
- read API authentication credentials from file. Allows mounting K8s secrets containing sensitive data
- upgraded outdated dependencies
- removed use of `promu`
- Dockerfile to build and run exporter

## Usage

Specify api key and secret:

```bash
nexmo_exporter --web.telemetry-port=":..." web.telemetry-path="https://..." --nexmo.url="https://..." --nexmo.timeout=5s --nexmo.namespace="/filepath"
```

### Launch as Docker container

```bash
docker run --name nexmo-exporter -p 9100:9100 \
-v /hostpath/credentials:/app/credentials:ro \
-e "PROMETHEUS_METRICS_PORT=9100" \
-e "PROMETHEUS_METRICS_PATH=/metrics" \
-e "NEXMO_URL=https://rest.nexmo.com" \
-e "NEXMO_TIMEOUT=5s" \
-e "NEXMO_PROMETHEUS_NAMESPACE=nexmo" \
g30rg3-stoica/nexmo-exporter
```

## Building the binary

```bash
make build
```

### Building the Docker image

```bash
docker build -t nexmo-exporter:latest .
```

## License

Apache License 2.0, see [LICENSE](https://github.com/prometheus/haproxy_exporter/blob/master/LICENSE).