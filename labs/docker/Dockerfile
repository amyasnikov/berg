FROM jauderho/gobgp:v3.37.0 AS fetcher


FROM golang:1.24-bookworm

WORKDIR /berg

COPY --from=fetcher /usr/local/bin/gobgp /bin/
COPY src/ /build/
COPY labs/docker/aliases.sh /tmp
RUN cd /build \
    && go build -o /berg/berg ./cmd \
    && cat /tmp/aliases.sh >> /etc/bash.bashrc

CMD ./berg -f config.toml
