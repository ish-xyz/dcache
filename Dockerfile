FROM golang:1.17
WORKDIR /mnt
ADD . /mnt/dcache
RUN cd /mnt/dcache  && \
    go mod tidy -compat=1.17 && \
    CGO_ENABLED=0 GOOS=linux go build -o dcache .

FROM bash:latest
RUN mkdir -p /var/dcache/data
COPY --from=0 /mnt/dcache/dcache /
ENTRYPOINT ["/dcache"]
