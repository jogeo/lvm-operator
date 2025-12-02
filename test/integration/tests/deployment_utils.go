package tests

import (
	"fmt"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type deployment struct {
	name         string
	namespace    string
	template     string
	pvcName      string
	typepath     string // "mountPath" for filesystem volumes, "devicePath" for block volumes
	replicas     string
	matchLabel   string
	testFileName string
}

// deploymentOption is a function option for configuring deployment parameters
type deploymentOption func(*deployment)

// setDeploymentName sets the deployment name
func setDeploymentName(name string) deploymentOption {
	return func(d *deployment) {
		d.name = name
	}
}

// setDeploymentTemplate sets the deployment template file path
func setDeploymentTemplate(template string) deploymentOption {
	return func(d *deployment) {
		d.template = template
	}
}

// setDeploymentNamespace sets the deployment namespace
func setDeploymentNamespace(namespace string) deploymentOption {
	return func(d *deployment) {
		d.namespace = namespace
	}
}

// setDeploymentPVCName sets the PVC name for the deployment
func setDeploymentPVCName(pvcName string) deploymentOption {
	return func(d *deployment) {
		d.pvcName = pvcName
	}
}

// setDeploymentTypePath sets the volume mount type ("mountPath" or "devicePath")
func setDeploymentTypePath(typepath string) deploymentOption {
	return func(d *deployment) {
		d.typepath = typepath
	}
}

// setDeploymentReplicas sets the number of replicas
func setDeploymentReplicas(replicas string) deploymentOption {
	return func(d *deployment) {
		d.replicas = replicas
	}
}

// setDeploymentMatchLabel sets the label selector
func setDeploymentMatchLabel(matchLabel string) deploymentOption {
	return func(d *deployment) {
		d.matchLabel = matchLabel
	}
}

// newDeployment creates a new deployment object with default values
func newDeployment(opts ...deploymentOption) deployment {
	defaultDeployment := deployment{
		name:         "my-dep-" + getRandomString(),
		namespace:    "",
		template:     "dep-template.yaml",
		pvcName:      "",
		typepath:     "mountPath", // Default to filesystem mount
		replicas:     "1",
		matchLabel:   "",
		testFileName: "testfile",
	}

	for _, o := range opts {
		o(&defaultDeployment)
	}

	// Set match label to deployment name if not specified
	if defaultDeployment.matchLabel == "" {
		defaultDeployment.matchLabel = defaultDeployment.name
	}

	return defaultDeployment
}

// create creates a new deployment resource
func (dep *deployment) create(tc *TestClient) {
	if dep.namespace == "" {
		dep.namespace = tc.Namespace()
	}

	var volumePath string
	if dep.typepath == "mountPath" {
		volumePath = "/mnt/test"
	} else {
		volumePath = "/dev/xvda"
	}

	err := applyResourceFromTemplate(tc, "--ignore-unknown-parameters=true", "-f", dep.template,
		"-p", "DEPNAME="+dep.name, "DEPNAMESPACE="+dep.namespace, "PVCNAME="+dep.pvcName,
		"REPLICAS="+dep.replicas, "MATCHLABEL="+dep.matchLabel, "VOLUMEPATH="+volumePath,
		"TYPEPATH="+dep.typepath)
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("Deployment %s created in namespace %s", dep.name, dep.namespace)
}

// delete deletes the deployment
func (dep *deployment) delete(tc *TestClient) {
	err := tc.WithoutNamespace().Run("delete").Args("deployment", dep.name, "-n", dep.namespace, "--ignore-not-found").Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("Deployment %s deleted from namespace %s", dep.name, dep.namespace)
}

// deleteAsAdmin deletes the deployment using admin credentials
func (dep *deployment) deleteAsAdmin(tc *TestClient) {
	tc.AsAdmin().WithoutNamespace().Run("delete").Args("deployment", dep.name, "-n", dep.namespace, "--ignore-not-found").Execute()
}

// waitReady waits for the deployment to be ready
func (dep *deployment) waitReady(tc *TestClient) {
	err := wait.Poll(10*time.Second, defaultMaxWaitingTime, func() (bool, error) {
		readyReplicas, err := tc.WithoutNamespace().Run("get").Args("deployment", dep.name, "-n", dep.namespace, "-o=jsonpath={.status.readyReplicas}").Output()
		if err != nil {
			e2e.Logf("Failed to get deployment %s status: %v, retrying...", dep.name, err)
			return false, nil
		}

		if readyReplicas == dep.replicas {
			e2e.Logf("Deployment %s is ready with %s replica(s)", dep.name, readyReplicas)
			return true, nil
		}

		e2e.Logf("Deployment %s: ready replicas %s, desired replicas %s", dep.name, readyReplicas, dep.replicas)
		return false, nil
	})

	if err != nil {
		// Log deployment details on failure
		output, _ := tc.WithoutNamespace().Run("describe").Args("deployment", dep.name, "-n", dep.namespace).Output()
		e2e.Logf("Deployment %s description:\n%s", dep.name, output)
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

// getPodName gets the first pod name for this deployment
func (dep *deployment) getPodName(tc *TestClient) string {
	podName, err := tc.WithoutNamespace().Run("get").Args("pods", "-n", dep.namespace, "-l", "app="+dep.matchLabel, "-o=jsonpath={.items[0].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(podName).NotTo(o.BeEmpty(), "No pod found for deployment "+dep.name)
	e2e.Logf("Pod name for deployment %s: %s", dep.name, podName)
	return podName
}

// checkPodMountedVolumeCouldRW checks that the mounted volume is readable and writable
func (dep *deployment) checkPodMountedVolumeCouldRW(tc *TestClient) {
	podName := dep.getPodName(tc)
	testFile := fmt.Sprintf("/mnt/test/%s", dep.testFileName)
	testContent := "test-data-" + getRandomString()

	// Write test data
	writeCmd := fmt.Sprintf("echo '%s' > %s", testContent, testFile)
	_, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", writeCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("Successfully wrote test data to %s in pod %s", testFile, podName)

	// Read and verify test data
	readCmd := fmt.Sprintf("cat %s", testFile)
	output, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", readCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(strings.TrimSpace(output)).To(o.Equal(testContent), "Read data does not match written data")
	e2e.Logf("Successfully verified read/write on volume in pod %s", podName)
}

// checkPodMountedVolumeDataExist checks if previously written data exists
func (dep *deployment) checkPodMountedVolumeDataExist(tc *TestClient, shouldExist bool) {
	podName := dep.getPodName(tc)
	testFile := fmt.Sprintf("/mnt/test/%s", dep.testFileName)

	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not-exists'", testFile)
	output, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", checkCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())

	output = strings.TrimSpace(output)
	if shouldExist {
		o.Expect(output).To(o.Equal("exists"), "Expected test file to exist but it doesn't")
		e2e.Logf("Test file %s exists in pod %s as expected", testFile, podName)
	} else {
		o.Expect(output).To(o.Equal("not-exists"), "Expected test file to not exist but it does")
		e2e.Logf("Test file %s does not exist in pod %s as expected", testFile, podName)
	}
}

// checkDataBlockType checks data on a block device
func (dep *deployment) checkDataBlockType(tc *TestClient) {
	podName := dep.getPodName(tc)
	devicePath := "/dev/xvda"

	// Read first 512 bytes and check for test marker
	readCmd := fmt.Sprintf("dd if=%s bs=512 count=1 2>/dev/null | grep -q 'TEST-BLOCK-DATA' && echo 'found' || echo 'not-found'", devicePath)
	output, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", readCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())

	output = strings.TrimSpace(output)
	o.Expect(output).To(o.Equal("found"), "Expected to find test marker on block device")
	e2e.Logf("Successfully verified block device data in pod %s", podName)
}

// writeDataBlockType writes test data to a block device
func (dep *deployment) writeDataBlockType(tc *TestClient) {
	podName := dep.getPodName(tc)
	devicePath := "/dev/xvda"

	// Write test marker to block device
	testMarker := "TEST-BLOCK-DATA-" + getRandomString()
	writeCmd := fmt.Sprintf("echo '%s' | dd of=%s bs=512 count=1 2>/dev/null", testMarker, devicePath)
	_, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", writeCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("Successfully wrote test data to block device in pod %s", podName)

	// Verify the write
	readCmd := fmt.Sprintf("dd if=%s bs=512 count=1 2>/dev/null | head -c %d", devicePath, len(testMarker))
	output, err := tc.WithoutNamespace().Run("exec").Args(podName, "-n", dep.namespace, "--", "sh", "-c", readCmd).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(strings.TrimSpace(output)).To(o.ContainSubstring("TEST-BLOCK-DATA"), "Written data verification failed")
	e2e.Logf("Successfully verified block device write in pod %s", podName)
}
