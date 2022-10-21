FROM scratch as builder

WORKDIR /
COPY bin/rio-csi bin/rio-csi

FROM ubuntu:18.04
WORKDIR /
COPY --from=builder /bin/rio-csi .
USER 65532:65532

ENTRYPOINT ["/rio-csi"]