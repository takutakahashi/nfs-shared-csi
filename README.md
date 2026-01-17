# nfs-shared-csi

A Kubernetes CSI driver for mounting NFS shares with support for ReadWriteMany (RWX) and ReadOnlyMany (ROX) access modes.

## Features

- Static provisioning of NFS volumes
- Support for RWX (ReadWriteMany) access mode
- Support for ROX (ReadOnlyMany) access mode
- NFS server and share path configured via StorageClass or PersistentVolume
- Configurable mount options

## Requirements

- Kubernetes 1.20+
- NFS client utilities installed on all nodes (`nfs-utils` or `nfs-common`)

## Installation

### Using Helm (Recommended)

```bash
helm install nfs-csi oci://ghcr.io/takutakahashi/charts/nfs-shared-csi \
  --extra-create-metadata
```

### Using kubectl

#### Build the image

```bash
make image
make push
```

#### Deploy to Kubernetes

```bash
kubectl apply -f deploy/kubernetes/
```

### Verify installation

```bash
kubectl get csidrivers
kubectl get pods -n kube-system -l app=csi-nfs-node
```

## Usage

### 1. Create a StorageClass

Edit `deploy/kubernetes/storageclass.yaml` with your NFS server details:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-csi
provisioner: nfs.csi.takutakahashi.dev
parameters:
  server: "192.168.1.100"    # Your NFS server address
  share: "/exports/data"     # Your NFS export path
mountOptions:
  - nfsvers=4.1
```

### 2. Create a PersistentVolume

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nfs-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteMany  # or ReadOnlyMany
  storageClassName: nfs-csi
  csi:
    driver: nfs.csi.takutakahashi.dev
    volumeHandle: unique-volume-id
    volumeAttributes:
      server: "192.168.1.100"
      share: "/exports/data"
```

### 3. Create a PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: nfs-csi
  resources:
    requests:
      storage: 10Gi
  volumeName: nfs-pv
```

### 4. Use in a Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
    - name: app
      image: busybox
      volumeMounts:
        - name: nfs-volume
          mountPath: /mnt/nfs
  volumes:
    - name: nfs-volume
      persistentVolumeClaim:
        claimName: nfs-pvc
```

## Access Modes

| Access Mode | Description |
|-------------|-------------|
| ReadWriteMany (RWX) | Multiple pods can read and write simultaneously |
| ReadOnlyMany (ROX) | Multiple pods can read simultaneously (read-only) |

## Configuration

### StorageClass Parameters

| Parameter | Description | Required |
|-----------|-------------|----------|
| `server` | NFS server address | Yes |
| `share` | NFS export path | Yes |

### Mount Options

Common mount options:
- `nfsvers=4.1` - NFS version
- `hard` - Hard mount (retry indefinitely)
- `soft` - Soft mount (fail after retries)
- `timeo=600` - Timeout in deciseconds
- `retrans=3` - Number of retries

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

## License

Apache License 2.0
