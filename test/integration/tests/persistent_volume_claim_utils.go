package tests

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type persistentVolumeClaim struct {
	name             string
	namespace        string
	scname           string
	template         string
	volumemode       string
	accessmode       string
	capacity         string
	dataSourceName   string
	maxWaitReadyTime time.Duration
}

// function option mode to change the default values of PersistentVolumeClaim parameters, e.g. name, namespace, accessmode, capacity, volumemode etc.
type persistentVolumeClaimOption func(*persistentVolumeClaim)

// Replace the default value of PersistentVolumeClaim name parameter
func setPersistentVolumeClaimName(name string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.name = name
	}
}

// Replace the default value of PersistentVolumeClaim template parameter
func setPersistentVolumeClaimTemplate(template string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.template = template
	}
}

// Replace the default value of PersistentVolumeClaim namespace parameter
func setPersistentVolumeClaimNamespace(namespace string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.namespace = namespace
	}
}

// Replace the default value of PersistentVolumeClaim accessmode parameter
func setPersistentVolumeClaimAccessmode(accessmode string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.accessmode = accessmode
	}
}

// Replace the default value of PersistentVolumeClaim scname parameter
func setPersistentVolumeClaimStorageClassName(scname string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.scname = scname
	}
}

// Replace the default value of PersistentVolumeClaim capacity parameter
func setPersistentVolumeClaimCapacity(capacity string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.capacity = capacity
	}
}

// Replace the default value of PersistentVolumeClaim volumemode parameter
func setPersistentVolumeClaimVolumemode(volumemode string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.volumemode = volumemode
	}
}

// Replace the default value of PersistentVolumeClaim DataSource Name
func setPersistentVolumeClaimDataSourceName(name string) persistentVolumeClaimOption {
	return func(this *persistentVolumeClaim) {
		this.dataSourceName = name
	}
}

// Create a new customized PersistentVolumeClaim object
func newPersistentVolumeClaim(opts ...persistentVolumeClaimOption) persistentVolumeClaim {
	defaultPersistentVolumeClaim := persistentVolumeClaim{
		name:             "my-pvc-" + getRandomString(),
		template:         "pvc-template.yaml",
		namespace:        "",
		capacity:         getValidVolumeSize(),
		volumemode:       "Filesystem",
		scname:           "gp2-csi",
		accessmode:       "ReadWriteOnce",
		maxWaitReadyTime: defaultMaxWaitingTime,
	}

	for _, o := range opts {
		o(&defaultPersistentVolumeClaim)
	}

	return defaultPersistentVolumeClaim
}

// Create new PersistentVolumeClaim with customized parameters
func (pvc *persistentVolumeClaim) create(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	err := applyResourceFromTemplate(tc, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Create new PersistentVolumeClaim without volumeMode
func (pvc *persistentVolumeClaim) createWithoutVolumeMode(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	o.Expect(applyResourceFromTemplateWithMultiExtraParameters(tc, []map[string]string{{"items.0.spec.volumeMode": "delete"}}, []map[string]interface{}{}, "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname, "ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)).Should(o.ContainSubstring("created"))
}

// Create new PersistentVolumeClaim with customized parameters to expect Error to occur
func (pvc *persistentVolumeClaim) createToExpectError(tc *TestClient) string {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	output, err := applyResourceFromTemplateWithOutput(tc, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).Should(o.HaveOccurred())
	return output
}

// Create a new PersistentVolumeClaim with clone dataSource parameters
func (pvc *persistentVolumeClaim) createWithCloneDataSource(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	dataSource := map[string]string{
		"kind": "PersistentVolumeClaim",
		"name": pvc.dataSourceName,
	}
	extraParameters := map[string]interface{}{
		"jsonPath":   `items.0.spec.`,
		"dataSource": dataSource,
	}
	err := applyResourceFromTemplateWithExtraParametersAsAdmin(tc, extraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Create a new PersistentVolumeClaim with clone dataSource parameters and null volumeMode
func (pvc *persistentVolumeClaim) createWithCloneDataSourceWithoutVolumeMode(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	dataSource := map[string]interface{}{
		"kind": "PersistentVolumeClaim",
		"name": pvc.dataSourceName,
	}
	jsonPathsAndActions := []map[string]string{{"items.0.spec.volumeMode": "delete"}, {"items.0.spec.dataSource.": "set"}}
	multiExtraParameters := []map[string]interface{}{{}, dataSource}
	o.Expect(applyResourceFromTemplateWithMultiExtraParameters(tc, jsonPathsAndActions, multiExtraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "PVCCAPACITY="+pvc.capacity)).Should(o.ContainSubstring("created"))
}

// Create a new PersistentVolumeClaim with snapshot dataSource parameters
func (pvc *persistentVolumeClaim) createWithSnapshotDataSource(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	dataSource := map[string]string{
		"kind":     "VolumeSnapshot",
		"name":     pvc.dataSourceName,
		"apiGroup": "snapshot.storage.k8s.io",
	}
	extraParameters := map[string]interface{}{
		"jsonPath":   `items.0.spec.`,
		"dataSource": dataSource,
	}
	err := applyResourceFromTemplateWithExtraParametersAsAdmin(tc, extraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Create a new PersistentVolumeClaim with custom dataSourceRef parameters
func (pvc *persistentVolumeClaim) createWithCustomDataSourceRef(tc *TestClient, dataSourceRef map[string]string) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	extraParameters := map[string]interface{}{
		"jsonPath":      `items.0.spec.`,
		"dataSourceRef": dataSourceRef,
	}
	err := applyResourceFromTemplateWithExtraParametersAsAdmin(tc, extraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Create a new PersistentVolumeClaim with specified persist volume
func (pvc *persistentVolumeClaim) createWithSpecifiedPV(tc *TestClient, pvName string) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	extraParameters := map[string]interface{}{
		"jsonPath":   `items.0.spec.`,
		"volumeName": pvName,
	}
	err := applyResourceFromTemplateWithExtraParametersAsAdmin(tc, extraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Create a new PersistentVolumeClaim without specifying storageClass name
func (pvc *persistentVolumeClaim) createWithoutStorageclassname(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}

	deletePaths := []string{`items.0.spec.storageClassName`}
	if isMicroshiftCluster(tc) {
		deletePaths = []string{`spec.storageClassName`}
	}
	err := applyResourceFromTemplateDeleteParametersAsAdmin(tc, deletePaths, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// create multiple PersistentVolumeClaim
func createMulPVC(tc *TestClient, begin int64, length int64, pvcTemplate string, storageClassName string) []persistentVolumeClaim {
	provisioner, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("storageclass/"+storageClassName, "-o", "jsonpath={.provisioner}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	provisionerBrief := strings.Split(provisioner, ".")[len(strings.Split(provisioner, "."))-2]
	var pvclist []persistentVolumeClaim
	for i := begin; i < begin+length+1; i++ {
		pvcname := "my-pvc-" + provisionerBrief + "-" + strconv.FormatInt(i, 10)
		pvclist = append(pvclist, newPersistentVolumeClaim(setPersistentVolumeClaimTemplate(pvcTemplate), setPersistentVolumeClaimName(pvcname), setPersistentVolumeClaimStorageClassName(storageClassName)))
		pvclist[i].create(tc)
	}
	return pvclist
}

// Create a new PersistentVolumeClaim with specified Volume Attributes Class (VAC)
func (pvc *persistentVolumeClaim) createWithSpecifiedVAC(tc *TestClient, vacName string) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	extraParameters := map[string]interface{}{
		"jsonPath":                  `items.0.spec.`,
		"volumeAttributesClassName": vacName,
	}
	err := applyResourceFromTemplateWithExtraParametersAsAdmin(tc, extraParameters, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Delete the PersistentVolumeClaim
func (pvc *persistentVolumeClaim) delete(tc *TestClient) {
	err := tc.WithoutNamespace().Run("delete").Args("pvc", pvc.name, "-n", pvc.namespace).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Delete the PersistentVolumeClaim use kubeadmin
func (pvc *persistentVolumeClaim) deleteAsAdmin(tc *TestClient) {
	tc.WithoutNamespace().AsAdmin().Run("delete").Args("pvc", pvc.name, "-n", pvc.namespace, "--ignore-not-found").Execute()
}

// Delete the PersistentVolumeClaim wait until timeout in seconds
func (pvc *persistentVolumeClaim) deleteUntilTimeOut(tc *TestClient, timeoutSeconds string) error {
	return tc.WithoutNamespace().Run("delete").Args("pvc", pvc.name, "-n", pvc.namespace, "--ignore-not-found", "--timeout="+timeoutSeconds+"s").Execute()
}

// Get the PersistentVolumeClaim status
func (pvc *persistentVolumeClaim) getStatus(tc *TestClient) (string, error) {
	pvcStatus, err := tc.WithoutNamespace().Run("get").Args("pvc", "-n", pvc.namespace, pvc.name, "-o=jsonpath={.status.phase}").Output()
	e2e.Logf("The PVC  %s status in namespace %s is %q", pvc.name, pvc.namespace, pvcStatus)
	return pvcStatus, err
}

// Get the PersistentVolumeClaim bounded  PersistentVolume's name
func (pvc *persistentVolumeClaim) getVolumeName(tc *TestClient) string {
	pvName, err := tc.WithoutNamespace().Run("get").Args("pvc", "-n", pvc.namespace, pvc.name, "-o=jsonpath={.spec.volumeName}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The PVC  %s in namespace %s Bound pv is %q", pvc.name, pvc.namespace, pvName)
	return pvName
}

// Get the PersistentVolumeClaim bounded  PersistentVolume's volumeID
func (pvc *persistentVolumeClaim) getVolumeID(tc *TestClient) string {
	pvName := pvc.getVolumeName(tc)
	volumeID, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pv", pvName, "-o=jsonpath={.spec.csi.volumeHandle}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The PV %s volumeID is %q", pvName, volumeID)
	return volumeID
}

// Get the description of PersistentVolumeClaim
func (pvc *persistentVolumeClaim) getDescription(tc *TestClient) (string, error) {
	output, err := tc.WithoutNamespace().Run("describe").Args("pvc", "-n", pvc.namespace, pvc.name).Output()
	e2e.Logf("****** The PVC  %s in namespace %s detail info: ******\n %s", pvc.name, pvc.namespace, output)
	return output, err
}

// Get the PersistentVolumeClaim bound pv's nodeAffinity nodeSelectorTerms matchExpressions "topology.gke.io/zone" values
func (pvc *persistentVolumeClaim) getVolumeNodeAffinityAvailableZones(tc *TestClient) []string {
	volName := pvc.getVolumeName(tc)
	return getPvNodeAffinityAvailableZones(tc, volName)
}

// Expand the PersistentVolumeClaim capacity, e.g. expandCapacity string "10Gi"
func (pvc *persistentVolumeClaim) expand(tc *TestClient, expandCapacity string) {
	expandPatchPath := "{\"spec\":{\"resources\":{\"requests\":{\"storage\":\"" + expandCapacity + "\"}}}}"
	patchResourceAsAdmin(tc, pvc.namespace, "pvc/"+pvc.name, expandPatchPath, "merge")
	pvc.capacity = expandCapacity
}

// Get pvc.status.capacity.storage value, sometimes it is different from request one
func (pvc *persistentVolumeClaim) getSizeFromStatus(tc *TestClient) string {
	pvcSize, err := getVolSizeFromPvc(tc, pvc.name, pvc.namespace)
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The PVC %s status.capacity.storage is %s", pvc.name, pvcSize)
	return pvcSize
}

// Get the PersistentVolumeClaim bounded  PersistentVolume's LastPhaseTransitionTime value
func (pvc *persistentVolumeClaim) getVolumeLastPhaseTransitionTime(tc *TestClient) string {
	pvName := pvc.getVolumeName(tc)
	lastPhaseTransitionTime, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pv", pvName, "-o=jsonpath={.status.lastPhaseTransitionTime}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The PV's %s lastPhaseTransitionTime is %s", pvName, lastPhaseTransitionTime)
	return lastPhaseTransitionTime
}

// capacityToBytes parses the pvc capacity to the int64 bytes value
func (pvc *persistentVolumeClaim) capacityToBytes(tc *TestClient) int64 {
	return parseCapacityToBytes(pvc.capacity)
}

// Get specified PersistentVolumeClaim status
func getPersistentVolumeClaimStatus(tc *TestClient, namespace string, pvcName string) (string, error) {
	pvcStatus, err := tc.WithoutNamespace().Run("get").Args("pvc", "-n", namespace, pvcName, "-o=jsonpath={.status.phase}").Output()
	e2e.Logf("The PVC  %s status in namespace %s is %q", pvcName, namespace, pvcStatus)
	return pvcStatus, err
}

// Describe specified PersistentVolumeClaim
func describePersistentVolumeClaim(tc *TestClient, namespace string, pvcName string) (string, error) {
	output, err := tc.WithoutNamespace().Run("describe").Args("pvc", "-n", namespace, pvcName).Output()
	e2e.Logf("****** The PVC  %s in namespace %s detail info: ******\n %s", pvcName, namespace, output)
	return output, err
}

// getPersistentVolumeClaimConditionStatus gets specified PersistentVolumeClaim conditions status during Resize
func getPersistentVolumeClaimConditionStatus(tc *TestClient, namespace string, pvcName string, conditionType string) (string, error) {
	pvcStatus, err := tc.WithoutNamespace().Run("get").Args("pvc", pvcName, "-n", namespace, fmt.Sprintf(`-o=jsonpath={.status.conditions[?(@.type=="%s")].status}`, conditionType)).Output()
	e2e.Logf("The PVC  %s status in namespace %s is %q", pvcName, namespace, pvcStatus)
	return pvcStatus, err
}

// Apply the patch to Resize volume
func applyVolumeResizePatch(tc *TestClient, pvcName string, namespace string, volumeSize string) (string, error) {
	command1 := "{\"spec\":{\"resources\":{\"requests\":{\"storage\":\"" + volumeSize + "\"}}}}"
	command := []string{"pvc", pvcName, "-n", namespace, "-p", command1, "--type=merge"}
	e2e.Logf("The command is %s", command)
	msg, err := tc.AsAdmin().WithoutNamespace().Run("patch").Args(command...).Output()
	if err != nil {
		e2e.Logf("Execute command failed with err:%v .", err)
		return msg, err
	}
	e2e.Logf("The command executed successfully %s", command)
	o.Expect(err).NotTo(o.HaveOccurred())
	return msg, nil
}

// Use persistent volume claim name to get the volumeSize in status.capacity
func getVolSizeFromPvc(tc *TestClient, pvcName string, namespace string) (string, error) {
	volumeSize, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pvc", pvcName, "-n", namespace, "-o=jsonpath={.status.capacity.storage}").Output()
	e2e.Logf("The PVC %s volumesize is %s", pvcName, volumeSize)
	return volumeSize, err
}

// Wait for PVC Volume Size to get Resized
func (pvc *persistentVolumeClaim) waitResizeSuccess(tc *TestClient, expandedCapactiy string) {
	waitPVCVolSizeToGetResized(tc, pvc.namespace, pvc.name, expandedCapactiy)
}

// Resizes the volume and checks data integrity
func (pvc *persistentVolumeClaim) resizeAndCheckDataIntegrity(tc *TestClient, dep deployment, expandedCapacity string) {
	o.Expect(applyVolumeResizePatch(tc, pvc.name, pvc.namespace, expandedCapacity)).To(o.ContainSubstring("patched"))
	pvc.capacity = expandedCapacity

	waitPVVolSizeToGetResized(tc, pvc.namespace, pvc.name, pvc.capacity)
	pvc.waitResizeSuccess(tc, pvc.capacity)

	if dep.typepath == "mountPath" {
		dep.checkPodMountedVolumeDataExist(tc, true)
		dep.checkPodMountedVolumeCouldRW(tc)
	} else {
		dep.checkDataBlockType(tc)
		dep.writeDataBlockType(tc)
	}
}

// Get the VolumeMode expected to equal
func (pvc *persistentVolumeClaim) checkVolumeModeAsexpected(tc *TestClient, vm string) {
	pvcVM, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pvc", pvc.name, "-n", pvc.namespace, "-o=jsonpath={.spec.volumeMode}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The pvc.spec.volumeMode is %s", pvcVM)
	o.Expect(pvcVM).To(o.Equal(vm))
}

// Check the status as Expected
func (pvc *persistentVolumeClaim) checkStatusAsExpectedConsistently(tc *TestClient, status string) {
	pvc.waitStatusAsExpected(tc, status)
	o.Consistently(func() string {
		pvcState, _ := pvc.getStatus(tc)
		return pvcState
	}, 20*time.Second, 5*time.Second).Should(o.Equal(status))
}

// Wait for PVC capacity expand successfully
func waitPVCVolSizeToGetResized(tc *TestClient, namespace string, pvcName string, expandedCapactiy string) {
	err := wait.Poll(10*time.Second, 180*time.Second, func() (bool, error) {
		capacity, err := getVolSizeFromPvc(tc, pvcName, namespace)
		if err != nil {
			e2e.Logf("Err occurred: \"%v\", get PVC: \"%s\" capacity failed.", err, pvcName)
			return false, err
		}
		if capacity == expandedCapactiy {
			e2e.Logf("The PVC: \"%s\" capacity expand to \"%s\"", pvcName, capacity)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		describePersistentVolumeClaim(tc, namespace, pvcName)
	}
}

// waitPersistentVolumeClaimConditionStatusAsExpected waits for PVC Volume resize condition status as expected
func waitPersistentVolumeClaimConditionStatusAsExpected(tc *TestClient, namespace string, pvcName string, conditionType string, expectedConditionStatus string) {
	_ = wait.Poll(10*time.Second, 180*time.Second, func() (bool, error) {
		status, err := getPersistentVolumeClaimConditionStatus(tc, namespace, pvcName, conditionType)
		if err != nil {
			e2e.Logf("Failed to get pvc %q condition status %v , try again.", pvcName, err)
			return false, nil
		}
		if status == expectedConditionStatus {
			e2e.Logf("The pvc resize condition %q changed to expected status:%q", conditionType, expectedConditionStatus)
			return true, nil
		}
		return false, nil
	})
}

// Get pvc list using selector label
func getPvcListWithLabel(tc *TestClient, selectorLabel string) []string {
	pvcList, err := tc.WithoutNamespace().Run("get").Args("pvc", "-n", tc.Namespace(), "-l", selectorLabel, "-o=jsonpath={.items[*].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The pvc list is %s", pvcList)
	return strings.Split(pvcList, " ")
}

// Check pvc counts matches with expected number
func checkPvcNumWithLabel(tc *TestClient, selectorLabel string, expectednum string) bool {
	if strconv.Itoa(cap(getPvcListWithLabel(tc, selectorLabel))) == expectednum {
		e2e.Logf("The pvc counts matched to expected replicas number: %s ", expectednum)
		return true
	}
	e2e.Logf("The pvc counts did not matched to expected replicas number: %s", expectednum)
	return false
}

// Wait persistentVolumeClaim status becomes to expected status
func (pvc *persistentVolumeClaim) waitStatusAsExpected(tc *TestClient, expectedStatus string) {
	var (
		status string
		err    error
	)
	if expectedStatus == "deleted" {
		err = wait.Poll(pvc.maxWaitReadyTime/defaultIterationTimes, pvc.maxWaitReadyTime, func() (bool, error) {
			status, err = pvc.getStatus(tc)
			if err != nil && strings.Contains(interfaceToString(err), "not found") {
				e2e.Logf("The persist volume claim '%s' becomes to expected status: '%s' ", pvc.name, expectedStatus)
				return true, nil
			}
			e2e.Logf("The persist volume claim '%s' is not deleted yet", pvc.name)
			return false, nil
		})
	} else {
		err = wait.Poll(pvc.maxWaitReadyTime/defaultIterationTimes, pvc.maxWaitReadyTime, func() (bool, error) {
			status, err = pvc.getStatus(tc)
			if err != nil {
				e2e.Logf("Get persist volume claim '%s' status failed of: %v.", pvc.name, err)
				return false, err
			}
			if status == expectedStatus {
				e2e.Logf("The persist volume claim '%s' becomes to expected status: '%s' ", pvc.name, expectedStatus)
				return true, nil
			}
			return false, nil

		})
	}
	if err != nil {
		describePersistentVolumeClaim(tc, pvc.namespace, pvc.name)
	}
}

// Wait persistentVolumeClaim status reach to expected status after 30sec timer
func (pvc *persistentVolumeClaim) waitPvcStatusToTimer(tc *TestClient, expectedStatus string) {
	//Check the status after 30sec of time
	var (
		status string
		err    error
	)
	currentTime := time.Now()
	e2e.Logf("Current time before wait of 30sec: %s", currentTime.String())
	err = wait.Poll(30*time.Second, 60*time.Second, func() (bool, error) {
		currentTime := time.Now()
		e2e.Logf("Current time after wait of 30sec: %s", currentTime.String())
		status, err = pvc.getStatus(tc)
		if err != nil {
			e2e.Logf("Get persist volume claim '%s' status failed of: %v.", pvc.name, err)
			return false, err
		}
		if status == expectedStatus {
			e2e.Logf("The persist volume claim '%s' remained in the expected status '%s'", pvc.name, expectedStatus)
			return true, nil
		}
		describePersistentVolumeClaim(tc, pvc.namespace, pvc.name)
		return false, nil
	})
}

// Get valid random capacity by volume type
func getValidRandomCapacityByCsiVolType(csiProvisioner string, volumeType string) string {
	var validRandomCapacityInt64 int64
	switch csiProvisioner {
	// aws-ebs-csi
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html
	// io1, io2, gp2, gp3, sc1, st1,standard
	// Default is gp3 if not set the volumeType in storageClass parameters
	case ebsCsiDriverProvisioner:
		// General Purpose SSD: 1 GiB - 16 TiB
		ebsGeneralPurposeSSD := []string{"gp2", "gp3"}
		// Provisioned IOPS SSD 4 GiB - 16 TiB
		ebsProvisionedIopsSSD := []string{"io1", "io2"}
		// HDD: {"sc1", "st1" 125 GiB - 16 TiB}, {"standard" 1 GiB - 1 TiB}
		ebsHDD := []string{"sc1", "st1", "standard"}

		if strSliceContains(ebsGeneralPurposeSSD, volumeType) || volumeType == "standard" {
			validRandomCapacityInt64 = getRandomNum(1, 10)
			break
		}
		if strSliceContains(ebsProvisionedIopsSSD, volumeType) {
			validRandomCapacityInt64 = getRandomNum(4, 20)
			break
		}
		if strSliceContains(ebsHDD, volumeType) && volumeType != "standard" {
			validRandomCapacityInt64 = getRandomNum(125, 200)
			break
		}
		validRandomCapacityInt64 = getRandomNum(1, 10)
	// aws-efs-csi
	// https://github.com/kubernetes-sigs/aws-efs-csi-driver
	// Actually for efs-csi volumes the capacity is meaningless
	// efs provides volumes almost unlimited capacity only billed by usage
	case efsCsiDriverProvisioner:
		validRandomCapacityInt64 = getRandomNum(1, 10)
	default:
		validRandomCapacityInt64 = getRandomNum(1, 10)
	}
	return strconv.FormatInt(validRandomCapacityInt64, 10) + "Gi"
}

// longerTime changes pvc.maxWaitReadyTime to specifiedDuring max wait time
// Used for some Longduration test
func (pvc *persistentVolumeClaim) specifiedLongerTime(specifiedDuring time.Duration) *persistentVolumeClaim {
	newPVC := *pvc
	newPVC.maxWaitReadyTime = specifiedDuring
	return &newPVC
}

// Patch PVC resource with the VolumeAttributesClass
func applyVolumeAttributesClassPatch(tc *TestClient, pvcName string, namespace string, vacName string) (string, error) {
	command1 := "{\"spec\":{\"volumeAttributesClassName\":\"" + vacName + "\"}}"
	command := []string{"pvc", pvcName, "-n", namespace, "-p", command1, "--type=merge"}
	e2e.Logf("The command is %s", command)
	msg, err := tc.AsAdmin().WithoutNamespace().Run("patch").Args(command...).Output()
	if err != nil {
		e2e.Logf("Execute command failed with err:%v .", err)
		return msg, err
	}
	e2e.Logf("The command executed successfully %s", command)
	o.Expect(err).NotTo(o.HaveOccurred())
	return msg, nil
}

// Get the currentVolumeAttributesClassName from the PVC
func (pvc *persistentVolumeClaim) getCurrentVolumeAttributesClassName(tc *TestClient) string {
	currentVolumeAttributesClassName, err := tc.AsAdmin().Run("get").Args("pvc", pvc.name, "-n", pvc.namespace, "-o=jsonpath={.status.currentVolumeAttributesClassName}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The PVC's %s current VolumeAttributesClass name is %s", pvc.name, currentVolumeAttributesClassName)
	return currentVolumeAttributesClassName
}

// Check PVC & bound PV resource has expected VolumeAttributesClass
func (pvc *persistentVolumeClaim) checkVolumeAttributesClassAsExpected(tc *TestClient, vacName string) {
	o.Eventually(func() string {
		vac := pvc.getCurrentVolumeAttributesClassName(tc)
		return vac
	}, 60*time.Second, 5*time.Second).Should(o.Equal(vacName))
	o.Eventually(func() string {
		vac := getVolumeAttributesClassFromPV(tc, pvc.getVolumeName(tc))
		return vac
	}, 60*time.Second, 5*time.Second).Should(o.Equal(vacName))
}

// Modify the PVC with specified VolumeAttributesClass
func (pvc *persistentVolumeClaim) modifyWithVolumeAttributesClass(tc *TestClient, vacName string) {
	_, err := applyVolumeAttributesClassPatch(tc, pvc.name, pvc.namespace, vacName)
	o.Expect(err).NotTo(o.HaveOccurred())
}

type pvcTopInfo struct {
	Namespace string
	Name      string
	Usage     string
}

// function to get all PersistentVolumeClaim Top info
func getPersistentVolumeClaimTopInfo(tc *TestClient) []pvcTopInfo {
	var pvcData []pvcTopInfo
	var pvcOutput string

	o.Eventually(func() string {
		pvcOutput, _ = tc.WithoutNamespace().Run("adm").Args("-n", tc.Namespace(), "top", "pvc", "--insecure-skip-tls-verify=true").Output()
		return pvcOutput
	}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).Should(o.ContainSubstring("USAGE"))
	pvcOutputLines := strings.Split(pvcOutput, "\n")
	space := regexp.MustCompile(`\s+`)
	for index, pvcOutputLine := range pvcOutputLines {
		if index == 0 {
			continue
		}
		fields := space.Split(pvcOutputLine, -1)
		if len(fields) >= 3 {
			pvc := pvcTopInfo{
				Namespace: fields[0],
				Name:      fields[1],
				Usage:     fields[2],
			}
			pvcData = append(pvcData, pvc)
		}
	}
	return pvcData
}

// function to wait till no pvc left
func checkZeroPersistentVolumeClaimTopInfo(tc *TestClient) {
	o.Eventually(func() string {
		pvcOutput, _ := tc.WithoutNamespace().Run("adm").Args("-n", tc.Namespace(), "top", "pvc", "--insecure-skip-tls-verify=true").Output()
		return pvcOutput
	}, 180*time.Second, 10*time.Second).Should(o.ContainSubstring("no persistentvolumeclaims found in use"))
}
