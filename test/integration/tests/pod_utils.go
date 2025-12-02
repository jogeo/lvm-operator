package tests

import (
	"strings"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// getPodsListByLabel returns a list of pod names in the given namespace matching the label selector
func getPodsListByLabel(tc *TestClient, namespace string, labelSelector string) ([]string, error) {
	output, err := tc.WithoutNamespace().Run("get").Args("pods", "-n", namespace, "-l", labelSelector, "-o=jsonpath={.items[*].metadata.name}").Output()
	if err != nil {
		e2e.Logf("Failed to get pods with label %q in namespace %q: %v", labelSelector, namespace, err)
		return nil, err
	}

	if output == "" {
		e2e.Logf("No pods found with label %q in namespace %q", labelSelector, namespace)
		return []string{}, nil
	}

	podsList := strings.Fields(output)
	e2e.Logf("Found %d pod(s) with label %q in namespace %q: %v", len(podsList), labelSelector, namespace, podsList)
	return podsList, nil
}

// waitPodReady waits for a pod to be ready
func waitPodReady(tc *TestClient, namespace string, podName string) {
	err := wait.Poll(10*time.Second, defaultMaxWaitingTime, func() (bool, error) {
		podStatus, err := tc.WithoutNamespace().Run("get").Args("pod", podName, "-n", namespace, "-o=jsonpath={.status.conditions[?(@.type=='Ready')].status}").Output()
		if err != nil {
			e2e.Logf("Failed to get pod %s status: %v, retrying...", podName, err)
			return false, nil
		}

		if podStatus == "True" {
			e2e.Logf("Pod %s in namespace %s is ready", podName, namespace)
			return true, nil
		}

		e2e.Logf("Pod %s in namespace %s is not ready yet, status: %s", podName, namespace, podStatus)
		return false, nil
	})

	o.Expect(err).NotTo(o.HaveOccurred())
}
