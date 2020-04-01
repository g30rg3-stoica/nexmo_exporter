FROM golang:1.13 AS build

ADD . /app
WORKDIR /app
RUN mkdir -p /app/bin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /app/bin/nexmo_exporter .

FROM golang:1.13.9-alpine
RUN mkdir -p app/credentials
COPY --from=build /app/bin/nexmo_exporter /app/nexmo_exporter
RUN chmod +x /app/nexmo_exporter

EXPOSE ${PROMETHEUS_METRICS_PORT}

ENTRYPOINT [ "sh", "-c", "/app/nexmo_exporter", \
"web.telemetry-port=:${PROMETHEUS_METRICS_PORT}", \
"web.telemetry-path=${PROMETHEUS_METRICS_PATH}", \
"nexmo.url=${NEXMO_URL}", \
"nexmo.timeout=${NEXMO_TIMEOUT}", \
"nexmo.namespace=${NEXMO_PROMETHEUS_NAMESPACE}" \
]
