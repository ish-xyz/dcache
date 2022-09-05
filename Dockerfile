FROM scratch

COPY ./dreg /dreg

ENTRYPOINT ["/dreg"]