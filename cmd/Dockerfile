FROM alpine:3.12
RUN apk add --no-cache lvm2 lvm2-extra util-linux device-mapper
RUN apk add --no-cache btrfs-progs xfsprogs xfsprogs-extra e2fsprogs e2fsprogs-extra
RUN apk add --no-cache ca-certificates libc6-compat

WORKDIR /
COPY bin/rio-csi bin/rio-csi
USER 65532:65532

ENTRYPOINT ["/bin/rio-csi"]