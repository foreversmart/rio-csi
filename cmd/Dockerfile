FROM alpine:3.12
RUN apk add --no-cache lvm2 lvm2-extra util-linux device-mapper
RUN apk add --no-cache btrfs-progs xfsprogs xfsprogs-extra e2fsprogs e2fsprogs-extra
RUN apk add --no-cache ca-certificates libc6-compat
RUN apk add --no-cache targetcli
RUN apk add --no-cache open-iscsi

WORKDIR /
COPY bin/rio-csi bin/rio-csi

ENTRYPOINT ["/bin/rio-csi"]