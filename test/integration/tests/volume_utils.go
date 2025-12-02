package tests

import (
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// getPvNodeAffinityAvailableZones gets the available zones from a PV's node affinity
func getPvNodeAffinityAvailableZones(tc *TestClient, pvName string) []string {
	// Get the PV node affinity zone information
	output, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pv", pvName, "-o=jsonpath={.spec.nodeAffinity.required.nodeSelectorTerms[*].matchExpressions[?(@.key=='topology.kubernetes.io/zone')].values}").Output()
	if err != nil || output == "" {
		e2e.Logf("Failed to get node affinity zones for PV %s: %v", pvName, err)
		return []string{}
	}

	// Parse the output - it's a JSON array-like string
	// Remove brackets and split by space
	output = strings.Trim(output, "[]")
	zones := strings.Fields(output)

	e2e.Logf("PV %s has node affinity zones: %v", pvName, zones)
	return zones
}

// waitPVVolSizeToGetResized waits for a PV to be resized to the expected capacity
func waitPVVolSizeToGetResized(tc *TestClient, namespace string, pvcName string, expectedCapacity string) {
	err := wait.Poll(10*time.Second, 180*time.Second, func() (bool, error) {
		// Get PV name from PVC
		pvName, err := tc.WithoutNamespace().Run("get").Args("pvc", pvcName, "-n", namespace, "-o=jsonpath={.spec.volumeName}").Output()
		if err != nil {
			e2e.Logf("Failed to get PV name for PVC %s: %v, retrying...", pvcName, err)
			return false, nil
		}

		// Get PV capacity
		pvCapacity, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pv", pvName, "-o=jsonpath={.spec.capacity.storage}").Output()
		if err != nil {
			e2e.Logf("Failed to get PV %s capacity: %v, retrying...", pvName, err)
			return false, nil
		}

		if pvCapacity == expectedCapacity {
			e2e.Logf("PV %s has been resized to %s", pvName, pvCapacity)
			return true, nil
		}

		e2e.Logf("PV %s capacity is %s, waiting for %s", pvName, pvCapacity, expectedCapacity)
		return false, nil
	})

	if err != nil {
		e2e.Logf("Timeout waiting for PV to resize for PVC %s", pvcName)
	}
}

// getVolumeAttributesClassFromPV gets the VolumeAttributesClass from a PV
func getVolumeAttributesClassFromPV(tc *TestClient, pvName string) string {
	vac, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("pv", pvName, "-o=jsonpath={.spec.volumeAttributesClassName}").Output()
	if err != nil {
		e2e.Logf("Failed to get VolumeAttributesClass from PV %s: %v", pvName, err)
		return ""
	}
	e2e.Logf("PV %s has VolumeAttributesClass: %s", pvName, vac)
	return vac
}
