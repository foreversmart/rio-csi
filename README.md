# rio-csi
Rio CSI is a standard K8s CSI Plugin which provide scalable, distributed
Persistent Storage. 
Rio-csi use LVM as it's persist storage implementation and iScsi to operate remote disks 

## Features

---
- [x] Access Modes
    - [x] ReadWriteOnce
    - [ ] ReadWriteMany
    - [ ] ReadOnlyMany
- [x] Volume modes
    - [x] `Filesystem` mode
    - [x] `Block` mode
- [x] Supports fsTypes: `ext4`, `btrfs`, `xfs`
- [ ] Volume metrics
- [x] Topology
- [x] Snapshot
- [ ] Clone
  - [x]from snapshot
  - [ ]from pvc
- [ ] Volume Resize
- [ ] Thin Provision
- [x] Backup/Restore
- [ ] Ephemeral inline volume

## Getting Started

### Install on the cluster
#### Prerequisites
* all the nodes must install lvm2 utils and load dm-snapshot kernel module for snapshot feature
```bash
yum install lvm2 -y # centos
sudo apt-get -y install lvm2 # ubuntu 

lsmod # 查看当前被内核加载的模块
cat /proc/modules # 也可以查看
# 通过 modinfo 查看 模块的详细信息
modinfo ext4
# modprobe 向内核增加或者删除指定的模块
modprobe btrfs # 增加 btrfs 模块
modprobe -r btrfs # 卸载 btrfs 模块

$ modprobe dm-snapshot # 或者  modprobe dm_snapshot
$ lsmod | grep dm
dm_snapshot            40960  0
dm_bufio               28672  1 dm_snapshot
rdma_cm                61440  1 ib_iser
iw_cm                  45056  1 rdma_cm
ib_cm                  53248  1 rdma_cm
ib_core               225280  4 rdma_cm,iw_cm,ib_iser,ib_cm
```
* Create LVM volume group on node, rio-csi will use this LVM volume group to manage storage
``` bash
# find available devices and partions
lsblk
# find available physical volumes
lvmdiskscan
# create pv
pvcreate /dev/test
# create volume group
vgcreate riovg /dev/test
```
#### Install Operator
* install open-iscsi on every node and make sure none of the node's InitiatorName are the same
```bash
# install iscsi 
apt -y install open-iscsi
# change to the same IQN you set on the iSCSI target server
vi /etc/iscsi/initiatorname.iscsi
InitiatorName=iqn.2018-05.world.srv:www.initiator01
```
* Install CSI Custom Resources and CSI Operator
```sh
kubectl apply -f operator.yaml
```
the default driver namespace is riocsi, all the RBAC resource and Operator is in riocsi

Verify that the rio-csi driver operator are installed and running using below command :
```
$ kubectl get pods -n rio-csi
NAME                               READY   STATUS    RESTARTS   AGE
csi-driver-node-ffgvv              2/2     Running   0          19s
csi-driver-node-hztjq              2/2     Running   0          19s
csi-driver-node-wsrtf              2/2     Running   0          19s
csi-provisioner-65b68bbcc8-b46rj   3/3     Running   0          19s
``
you cant find csi-driver-node daemonset running on every node in the cluster and one csi-provisioner

#### Snapshot feature

* Open K8s Snapshot feature
As snapshot is a beata feature of K8s we should open K8s snapshot feature config to use this feature
``` bash
kubectl --feature-gates VolumeSnapshotDataSource=true
```
* Install Snapshot Controller
K8s Cluster have no Snapshot Controller as default. Apply snapshotoperator.yaml to install snapshot controller
and snapshot CRDs
```bash
kubectl apply -f snapshotoperator.yaml
```


* Build and push your image to the location specified by `IMG`:

### How To Use
* Create a Storage Class to use this driver
```shell
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: rio-sc
parameters:
  storage: "lvm"
  volgroup: "myvg"
provisioner: rio-csi
```

Specify volgroup to select Volume Group from nodes

* create PVC to use above Storage Class
```shell
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: test-pvc
spec:
  storageClassName: rio-sc
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 2G
```
* Check Volume is created
```shell
kubectl get volume -n riocsi
```

### Uninstall CRDs
To delete the CRDs from the cluster:
// TODO

### Undeploy controller
UnDeploy the controller to the cluster:
// TODO

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
// TODO
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 


## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

