FROM ubuntu:20.04
RUN apt update
RUN apt install -y lvm2
RUN apt install -y targetcli-fb
RUN apt install -y open-iscsi
RUN apt install -y kmod

WORKDIR /
COPY bin/rio-csi bin/rio-csi

ENTRYPOINT ["/bin/rio-csi"]