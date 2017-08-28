# Nexmo Exporter for Prometheus

[![Go Report Card](https://goreportcard.com/badge/github.com/pzinovkin/nexmo_exporter)](https://goreportcard.com/report/github.com/pzinovkin/nexmo_exporter)

Simple server that scraps Nexmo balance and exports it as Prometheus metrics.

## Usage

Specify api key and secret:

```bash
nexmo_exporter --nexmo.api-key="..." --nexmo.api-secret="..."
```


## Building

```bash
make build
```

## License

Apache License 2.0, see [LICENSE](https://github.com/prometheus/haproxy_exporter/blob/master/LICENSE).