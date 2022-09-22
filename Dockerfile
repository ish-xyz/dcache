FROM golang:1.17
WORKDIR /mnt
ADD . /mnt/dpc
RUN cd /mnt/dpc  && \
    go mod tidy -compat=1.17 && \
    CGO_ENABLED=0 GOOS=linux go build -o dpc .

FROM alpine:latest
RUN mkdir -p /var/dpc/data
COPY --from=0 /mnt/dpc/dpc /
ENTRYPOINT ["/dpc"]
