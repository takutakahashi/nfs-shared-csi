package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	driverName     = "nfs.csi.example.com"
	testNamespace  = "default"
	storageClass   = "nfs-csi-e2e"
	defaultTimeout = 5 * time.Minute
)

var (
	clientset *kubernetes.Clientset
	nfsServer string
	nfsShare  string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NFS CSI Driver E2E Suite")
}

var _ = BeforeSuite(func() {
	// Get NFS server config from environment
	nfsServer = os.Getenv("NFS_SERVER")
	nfsShare = os.Getenv("NFS_SHARE")

	if nfsServer == "" || nfsShare == "" {
		Skip("NFS_SERVER and NFS_SHARE environment variables must be set for E2E tests")
	}

	// Create Kubernetes client
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	clientset, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	// Create test StorageClass
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClass,
		},
		Provisioner: driverName,
		Parameters: map[string]string{
			"server": nfsServer,
			"share":  nfsShare,
		},
	}

	_, err = clientset.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
	if err != nil {
		// Ignore if already exists
		GinkgoWriter.Printf("StorageClass creation: %v\n", err)
	}
})

var _ = AfterSuite(func() {
	if clientset != nil {
		// Cleanup StorageClass
		_ = clientset.StorageV1().StorageClasses().Delete(context.TODO(), storageClass, metav1.DeleteOptions{})
	}
})

var _ = Describe("NFS CSI Driver", func() {
	var (
		pvName  string
		pvcName string
		podName string
	)

	BeforeEach(func() {
		// Generate unique names for each test
		suffix := fmt.Sprintf("%d", time.Now().UnixNano())
		pvName = "nfs-pv-" + suffix[:8]
		pvcName = "nfs-pvc-" + suffix[:8]
		podName = "nfs-pod-" + suffix[:8]
	})

	AfterEach(func() {
		// Cleanup resources
		ctx := context.TODO()
		_ = clientset.CoreV1().Pods(testNamespace).Delete(ctx, podName, metav1.DeleteOptions{})
		_ = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
		_ = clientset.CoreV1().PersistentVolumes().Delete(ctx, pvName, metav1.DeleteOptions{})
	})

	Context("ReadWriteMany (RWX)", func() {
		It("should mount NFS volume with RWX access mode", func() {
			ctx := context.TODO()

			// Create PV
			pv := createPV(pvName, nfsServer, nfsShare, corev1.ReadWriteMany, false)
			_, err := clientset.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create PVC
			pvc := createPVC(pvcName, pvName, corev1.ReadWriteMany)
			_, err = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(ctx, pvc, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create Pod
			pod := createPod(podName, pvcName, false)
			_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, pod, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for Pod to be running
			Eventually(func() corev1.PodPhase {
				p, err := clientset.CoreV1().Pods(testNamespace).Get(ctx, podName, metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return p.Status.Phase
			}, defaultTimeout, 5*time.Second).Should(Equal(corev1.PodRunning))

			// Verify mount is writable
			// (In a real test, you would exec into the pod and verify)
		})
	})

	Context("ReadOnlyMany (ROX)", func() {
		It("should mount NFS volume with ROX access mode", func() {
			ctx := context.TODO()

			// Create PV with readOnly
			pv := createPV(pvName, nfsServer, nfsShare, corev1.ReadOnlyMany, true)
			_, err := clientset.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create PVC
			pvc := createPVC(pvcName, pvName, corev1.ReadOnlyMany)
			_, err = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(ctx, pvc, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create Pod with readOnly mount
			pod := createPod(podName, pvcName, true)
			_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, pod, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for Pod to be running
			Eventually(func() corev1.PodPhase {
				p, err := clientset.CoreV1().Pods(testNamespace).Get(ctx, podName, metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return p.Status.Phase
			}, defaultTimeout, 5*time.Second).Should(Equal(corev1.PodRunning))
		})
	})

	Context("Multiple Pods", func() {
		It("should allow multiple pods to mount the same RWX volume", func() {
			ctx := context.TODO()
			pod2Name := podName + "-2"

			// Create PV
			pv := createPV(pvName, nfsServer, nfsShare, corev1.ReadWriteMany, false)
			_, err := clientset.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create PVC
			pvc := createPVC(pvcName, pvName, corev1.ReadWriteMany)
			_, err = clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(ctx, pvc, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create first Pod
			pod1 := createPod(podName, pvcName, false)
			_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, pod1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create second Pod
			pod2 := createPod(pod2Name, pvcName, false)
			_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, pod2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for both Pods to be running
			for _, name := range []string{podName, pod2Name} {
				Eventually(func() corev1.PodPhase {
					p, err := clientset.CoreV1().Pods(testNamespace).Get(ctx, name, metav1.GetOptions{})
					if err != nil {
						return ""
					}
					return p.Status.Phase
				}, defaultTimeout, 5*time.Second).Should(Equal(corev1.PodRunning))
			}

			// Cleanup second pod
			_ = clientset.CoreV1().Pods(testNamespace).Delete(ctx, pod2Name, metav1.DeleteOptions{})
		})
	})
})

func createPV(name, server, share string, accessMode corev1.PersistentVolumeAccessMode, readOnly bool) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{accessMode},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              storageClass,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       driverName,
					VolumeHandle: name,
					ReadOnly:     readOnly,
					VolumeAttributes: map[string]string{
						"server": server,
						"share":  share,
					},
				},
			},
		},
	}
}

func createPVC(name, pvName string, accessMode corev1.PersistentVolumeAccessMode) *corev1.PersistentVolumeClaim {
	scName := storageClass
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{accessMode},
			StorageClassName: &scName,
			VolumeName:       pvName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func createPod(name, pvcName string, readOnly bool) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "busybox:1.36",
					Command: []string{
						"sh", "-c", "sleep infinity",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "nfs-volume",
							MountPath: "/mnt/nfs",
							ReadOnly:  readOnly,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "nfs-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
}
