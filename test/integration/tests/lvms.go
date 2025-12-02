// Source: test/extended/storage/lvms.go from openshift-tests-private
// Test: Author:rdeore-LEVEL0-Critical-73162-[LVMS] Check LVMCluster works with the devices configured for both thin and thick provisioning [Disruptive]

package tests

import (
	"math"
	"path/filepath"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = g.Describe("[LVMS][Disruptive] LVMCluster", g.Label("LVMS", "Disruptive"), func() {
	defer g.GinkgoRecover()

	var (
		tc = NewTestClient("lvms-test")
	)

	// author: rdeore@redhat.com
	// OCP-73162-[LVMS] Check LVMCluster works with the devices configured for both thin and thick provisioning
	g.It("Author:rdeore-LEVEL0-Critical-73162-[LVMS] Check LVMCluster works with the devices configured for both thin and thick provisioning [Disruptive]", func() {
		// Set the resource template for the scenario
		var (
			pvcTemplate         = filepath.Join("testdata", "storage", "pvc-template.yaml")
			deploymentTemplate  = filepath.Join("testdata", "storage", "dep-template.yaml")
			lvmClusterTemplate  = filepath.Join("testdata", "storage", "lvms", "lvmcluster-with-multi-device-template.yaml")
			storageClass1       = "lvms-vg1"
			storageClass2       = "lvms-vg2"
			volumeSnapshotClass = "lvms-vg1"
		)

		g.By("#. Get list of available block devices/disks attached to all worker nodes")
		freeDiskNameCountMap := getListOfFreeDisksFromWorkerNodes(tc)
		if len(freeDiskNameCountMap) < 2 { // this test requires atleast 2 unique disks
			g.Skip("Skipped: Cluster's Worker nodes does not have minimum required free block devices/disks attached")
		}
		workerNodeCount := len(getWorkersList(tc))
		var devicePaths []string
		for diskName, count := range freeDiskNameCountMap {
			if count == int64(workerNodeCount) { // mandatory disk/device with same name present on all worker nodes
				devicePaths = append(devicePaths, "/dev/"+diskName)
				delete(freeDiskNameCountMap, diskName)
				if len(devicePaths) == 2 { // only two disks/devices are required
					break
				}
			}
		}
		if len(devicePaths) < 2 { // If all Worker nodes doesn't have atleast two free disks/devices with same name, skip the test scenario
			g.Skip("Skipped: All Worker nodes does not have atleast two required free block disks/devices with same name attached")
		}

		g.By("#. Copy and save existing LVMCluster configuration in JSON format")
		lvmClusterName, err := tc.AdminKubeClient().CoreV1().RESTClient().Get().AbsPath("/apis/lvm.topolvm.io/v1alpha1/namespaces/openshift-lvm-storage/lvmclusters").DoRaw(tc.Context())
		o.Expect(err).NotTo(o.HaveOccurred())
		originLvmCluster := newLvmCluster(setLvmClusterName(string(lvmClusterName)), setLvmClusterNamespace("openshift-lvm-storage"))
		originLVMJSON, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", originLvmCluster.name, "-n", "openshift-lvm-storage", "-o", "json").Output()
		e2e.Logf("Original LVMCluster saved")
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("#. Delete existing LVMCluster resource")
		deleteSpecifiedResource(tc.AsAdmin(), "lvmcluster", originLvmCluster.name, "openshift-lvm-storage")
		defer func() {
			if !isSpecifiedResourceExist(tc, "lvmcluster/"+originLvmCluster.name, "openshift-lvm-storage") {
				originLvmCluster.createWithExportJSON(tc, originLVMJSON, originLvmCluster.name)
			}
			originLvmCluster.waitReady(tc)
		}()

		g.By("#. Create a new LVMCluster resource with two device-classes")
		lvmCluster := newLvmCluster(setLvmClustertemplate(lvmClusterTemplate), setLvmClusterPaths([]string{devicePaths[0], devicePaths[1]}))
		lvmCluster.createWithMultiDeviceClasses(tc)
		defer lvmCluster.deleteLVMClusterSafely(tc) // If new lvmCluster creation fails, need to remove finalizers if any
		lvmCluster.waitReady(tc)

		g.By("#. Create a new project for the scenario")
		tc.SetupProject()

		g.By("Check two lvms preset storage-classes are present one for each volumeGroup")
		checkStorageclassExists(tc, storageClass1)
		checkStorageclassExists(tc, storageClass2)

		g.By("Check only one preset lvms volumeSnapshotClass is present for volumeGroup with thinPoolConfig")
		o.Expect(isSpecifiedResourceExist(tc, "volumesnapshotclass/"+volumeSnapshotClass, "")).To(o.BeTrue())
		o.Expect(isSpecifiedResourceExist(tc, "volumesnapshotclass/lvms-vg2", "")).To(o.BeFalse())

		g.By("Check available storage capacity of preset lvms SC (thick provisioning) equals to the backend total disks size")
		thickProvisioningStorageCapacity := lvmCluster.getCurrentTotalLvmStorageCapacityByStorageClass(tc, storageClass2) / 1024
		e2e.Logf("ACTUAL USABLE STORAGE CAPACITY: %d", thickProvisioningStorageCapacity)
		pathsDiskTotalSize := getTotalDiskSizeOnAllWorkers(tc, devicePaths[1])
		e2e.Logf("BACKEND DISK SIZE: %d", pathsDiskTotalSize)
		storageDiff := float64(thickProvisioningStorageCapacity - pathsDiskTotalSize)
		absDiff := math.Abs(storageDiff)
		o.Expect(int(absDiff) < 2).To(o.BeTrue()) // there is always a difference of 1 Gi between backend disk size and usable size

		g.By("#. Define storage resources")
		pvc1 := newPersistentVolumeClaim(setPersistentVolumeClaimTemplate(pvcTemplate), setPersistentVolumeClaimStorageClassName(storageClass1))
		pvc2 := newPersistentVolumeClaim(setPersistentVolumeClaimTemplate(pvcTemplate), setPersistentVolumeClaimCapacity("20Mi"),
			setPersistentVolumeClaimStorageClassName(storageClass2))
		dep1 := newDeployment(setDeploymentTemplate(deploymentTemplate), setDeploymentPVCName(pvc1.name), setDeploymentNamespace(tc.Namespace()))
		dep2 := newDeployment(setDeploymentTemplate(deploymentTemplate), setDeploymentPVCName(pvc2.name), setDeploymentNamespace(tc.Namespace()))

		g.By("#. Create a pvc-1 with the preset lvms csi storageclass with thin provisioning")
		pvc1.create(tc)
		defer pvc1.deleteAsAdmin(tc)

		g.By("#. Create a deployment-1 with the created pvc-1 and wait for the pod ready")
		dep1.create(tc)
		defer dep1.deleteAsAdmin(tc)
		dep1.waitReady(tc)

		g.By("#. Write a file to volume")
		dep1.checkPodMountedVolumeCouldRW(tc)

		g.By("#. Create a pvc-2 with the preset lvms csi storageclass with thick provisioning")
		pvc2.create(tc)
		defer pvc2.deleteAsAdmin(tc)

		g.By("#. Create a deployment-2 with the created pvc-2 and wait for the pod ready")
		dep2.create(tc)
		defer dep2.deleteAsAdmin(tc)
		dep2.waitReady(tc)

		g.By("#. Write a file to volume")
		dep2.checkPodMountedVolumeCouldRW(tc)

		g.By("#. Resize pvc-2 storage capacity to a value bigger than 1Gi")
		pvc2.resizeAndCheckDataIntegrity(tc, dep2, "2Gi")

		g.By("Delete Deployments and PVCs")
		deleteSpecifiedResource(tc, "deployment", dep1.name, dep1.namespace)
		deleteSpecifiedResource(tc, "pvc", pvc1.name, pvc1.namespace)
		deleteSpecifiedResource(tc, "deployment", dep2.name, dep2.namespace)
		deleteSpecifiedResource(tc, "pvc", pvc2.name, pvc2.namespace)

		g.By("Delete newly created LVMCluster resource")
		lvmCluster.deleteLVMClusterSafely(tc)

		g.By("#. Create original LVMCluster resource")
		originLvmCluster.createWithExportJSON(tc, originLVMJSON, originLvmCluster.name)
		originLvmCluster.waitReady(tc)
	})
})
