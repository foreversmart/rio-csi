FROM ubuntu:18.04
RUN apt update
RUN apt install -y lvm2
RUN apt install -y targetcli-fb
RUN apt install -y open-iscsi

WORKDIR /
COPY lvm.conf /etc/lvm/lvm.conf
COPY bin/rio-csi bin/rio-csi

ENTRYPOINT ["/bin/rio-csi"]