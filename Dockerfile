FROM golang:1.17
WORKDIR /mnt
ADD . /mnt/dreg
RUN cd /mnt/dreg  && \
    go mod tidy -compat=1.17 && \
    CGO_ENABLED=0 GOOS=linux go build -o dreg .

FROM alpine:latest
RUN mkdir -p /var/dreg/data
COPY --from=0 /mnt/dreg/dreg /
ENTRYPOINT ["/dreg"]
