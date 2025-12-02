package tests

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tidwall/sjson"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	// defaultMaxWaitingTime is the default maximum time to wait for operations to complete
	defaultMaxWaitingTime = 300 * time.Second

	// defaultIterationTimes is the default number of polling iterations
	defaultIterationTimes = 30

	// bytesPerGiB is the number of bytes in a gibibyte
	bytesPerGiB = 1024 * 1024 * 1024

	// ebsCsiDriverProvisioner is the provisioner name for AWS EBS CSI driver
	ebsCsiDriverProvisioner = "ebs.csi.aws.com"

	// efsCsiDriverProvisioner is the provisioner name for AWS EFS CSI driver
	efsCsiDriverProvisioner = "efs.csi.aws.com"
)

var (
	// cloudProvider indicates which cloud provider the cluster is running on
	// Can be set via CLOUD_PROVIDER environment variable
	cloudProvider = getCloudProvider()
)

// debugLogf logs debug messages in a conditional manner
// Only logs if debug mode is enabled to avoid test output noise
func debugLogf(format string, args ...interface{}) {
	// Debug logging is commented out by default to reduce noise
	// Uncomment the line below if you need detailed debug output
	// e2e.Logf("[DEBUG] "+format, args...)
	_ = format
	_ = args
}

// getRandomString generates a random string for unique resource names
func getRandomString() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "random"
	}
	return hex.EncodeToString(bytes)
}

// strSliceContains checks if a string slice contains a specific string
func strSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// getCloudProvider detects the cloud provider from environment or returns "unknown"
func getCloudProvider() string {
	provider := os.Getenv("CLOUD_PROVIDER")
	if provider == "" {
		// Try to detect from common environment variables
		if os.Getenv("AWS_REGION") != "" {
			return "aws"
		}
		if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
			return "azure"
		}
		if os.Getenv("GCP_PROJECT") != "" {
			return "gcp"
		}
		return "unknown"
	}
	return strings.ToLower(provider)
}

// deleteSpecifiedResource deletes a Kubernetes resource
func deleteSpecifiedResource(tc *TestClient, resourceType string, name string, namespace string) {
	args := []string{resourceType, name, "--ignore-not-found"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	tc.WithoutNamespace().Run("delete").Args(args...).Execute()
	e2e.Logf("Deleted %s/%s in namespace %s", resourceType, name, namespace)
}

// isSpecifiedResourceExist checks if a Kubernetes resource exists
// resourcePath should be in format "type/name" (e.g., "lvmcluster/my-cluster")
func isSpecifiedResourceExist(tc *TestClient, resourcePath string, namespace string) bool {
	args := []string{resourcePath, "--ignore-not-found"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	output, err := tc.WithoutNamespace().Run("get").Args(args...).Output()
	if err != nil || output == "" {
		return false
	}
	return true
}

// checkStorageclassExists checks if a storage class exists
func checkStorageclassExists(tc *TestClient, name string) {
	output, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("storageclass", name, "--ignore-not-found").Output()
	if err != nil || output == "" {
		e2e.Failf("StorageClass %s does not exist", name)
	}
	e2e.Logf("StorageClass %s exists", name)
}

// patchResourceAsAdmin patches a Kubernetes resource using admin credentials
// patchType can be "merge", "json", or "strategic"
func patchResourceAsAdmin(tc *TestClient, namespace string, resource string, patchData string, patchType string) {
	args := []string{resource, "-p", patchData, "--type=" + patchType}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	err := tc.AsAdmin().WithoutNamespace().Run("patch").Args(args...).Execute()
	if err != nil {
		e2e.Logf("Failed to patch %s: %v", resource, err)
	} else {
		e2e.Logf("Successfully patched %s in namespace %s", resource, namespace)
	}
}

// jsonAddExtraParametersToFile adds extra parameters to a JSON string
func jsonAddExtraParametersToFile(jsonData string, extraParams map[string]interface{}) (string, error) {
	// This function uses sjson to add/modify JSON fields
	result := jsonData
	var err error

	for key, value := range extraParams {
		result, err = sjson.Set(result, key, value)
		if err != nil {
			return "", err
		}
	}

	return result, nil
}

// getValidVolumeSize returns a valid default volume size
func getValidVolumeSize() string {
	return "1Gi"
}

// isMicroshiftCluster checks if the cluster is a MicroShift cluster
func isMicroshiftCluster(tc *TestClient) bool {
	// Check for MicroShift-specific indicators
	output, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion", "--ignore-not-found").Output()
	if err != nil || output == "" {
		// If clusterversion doesn't exist, might be MicroShift
		// Check for other indicators
		output, err = tc.AsAdmin().WithoutNamespace().Run("get").Args("nodes", "-l", "node-role.kubernetes.io/master", "-o=jsonpath={.items[*].metadata.name}").Output()
		if err == nil && len(strings.Fields(output)) == 1 {
			// Single master node might indicate MicroShift
			return true
		}
	}
	return false
}

// interfaceToString converts an interface to string
func interfaceToString(i interface{}) string {
	if i == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", i)))
}

// getRandomNum generates a random number in the range [min, max]
func getRandomNum(min, max int64) int64 {
	if min >= max {
		return min
	}
	bytes := make([]byte, 8)
	rand.Read(bytes)
	// Simple random number generation
	num := int64(bytes[0])%( max-min+1) + min
	return num
}

// parseCapacityToBytes parses a capacity string (e.g., "10Gi") to bytes
func parseCapacityToBytes(capacity string) int64 {
	capacity = strings.TrimSpace(capacity)
	if capacity == "" {
		return 0
	}

	// Remove the unit suffix and parse the number
	var multiplier int64 = 1
	if strings.HasSuffix(capacity, "Gi") || strings.HasSuffix(capacity, "GiB") {
		multiplier = bytesPerGiB
		capacity = strings.TrimSuffix(strings.TrimSuffix(capacity, "GiB"), "Gi")
	} else if strings.HasSuffix(capacity, "Mi") || strings.HasSuffix(capacity, "MiB") {
		multiplier = 1024 * 1024
		capacity = strings.TrimSuffix(strings.TrimSuffix(capacity, "MiB"), "Mi")
	} else if strings.HasSuffix(capacity, "Ki") || strings.HasSuffix(capacity, "KiB") {
		multiplier = 1024
		capacity = strings.TrimSuffix(strings.TrimSuffix(capacity, "KiB"), "Ki")
	} else if strings.HasSuffix(capacity, "G") {
		multiplier = 1000 * 1000 * 1000
		capacity = strings.TrimSuffix(capacity, "G")
	} else if strings.HasSuffix(capacity, "M") {
		multiplier = 1000 * 1000
		capacity = strings.TrimSuffix(capacity, "M")
	} else if strings.HasSuffix(capacity, "K") {
		multiplier = 1000
		capacity = strings.TrimSuffix(capacity, "K")
	}

	// Parse the numeric part
	var value int64
	fmt.Sscanf(capacity, "%d", &value)
	return value * multiplier
}
