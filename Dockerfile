FROM scratch

# get the binary
COPY passenger-datadog-monitor /usr/local/bin/passenger-datadog-monitor

WORKDIR /work

ENTRYPOINT ["/usr/local/bin/passenger-datadog-monitor"]
