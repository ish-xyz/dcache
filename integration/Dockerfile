FROM golang:1.17
WORKDIR /mnt
RUN git clone https://github.com/ish-xyz/dreg.git
RUN cd /mnt/dreg  && \
    go mod tidy -compat=1.17 && \
    CGO_ENABLED=0 GOOS=linux go build -o dreg .

FROM scratch
COPY --from=0 /mnt/dreg/dreg /
ENTRYPOINT ["/dreg"]