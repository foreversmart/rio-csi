FROM scratch as builder

WORKDIR /
COPY bin/manager bin/manager

FROM ubuntu:18.04
WORKDIR /
COPY --from=builder /bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]