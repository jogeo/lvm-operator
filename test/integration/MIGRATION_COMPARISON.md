# Migration Comparison: Real Examples from test/integration/tests

This document shows side-by-side comparisons of real functions from the codebase.

## Example 1: getLvmClusterStatus (lvms_utils.go:290)

### Before (using exutil.CLI)
```go
func (lvm *lvmCluster) getLvmClusterStatus(oc *exutil.CLI) (string, error) {
	lvmCluster, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", "-n", lvm.namespace, "-o", "json").Output()
	lvmClusterState := gjson.Get(lvmCluster, "items.#(metadata.name="+lvm.name+").status.state").String()
	e2e.Logf("The current LVM Cluster state is %q", lvmClusterState)
	return lvmClusterState, err
}
```

### After Option A (drop-in replacement)
```go
func (lvm *lvmCluster) getLvmClusterStatus(tc *TestClient) (string, error) {
	lvmCluster, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", "-n", lvm.namespace, "-o", "json").Output()
	lvmClusterState := gjson.Get(lvmCluster, "items.#(metadata.name="+lvm.name+").status.state").String()
	e2e.Logf("The current LVM Cluster state is %q", lvmClusterState)
	return lvmClusterState, err
}
```
**Change:** Just replace `oc *exutil.CLI` with `tc *TestClient`

### After Option B (type-safe - recommended)
```go
func (lvm *lvmCluster) getLvmClusterStatus(tc *TestClient) (string, error) {
	cluster, err := tc.GetLVMCluster(lvm.name, lvm.namespace)
	if err != nil {
		return "", err
	}

	lvmClusterState := string(cluster.Status.State)
	e2e.Logf("The current LVM Cluster state is %q", lvmClusterState)
	return lvmClusterState, nil
}
```
**Benefits:**
- No JSON parsing needed
- Type-safe field access
- Compiler catches typos
- Better IDE autocomplete

---

## Example 2: getCurrentLVMClusterName (lvms_utils.go:305)

### Before
```go
func getCurrentLVMClusterName(oc *exutil.CLI) string {
	output, err := oc.AsAdmin().Run("get").Args("lvmcluster", "-n", "openshift-lvm-storage", "-o=custom-columns=NAME:.metadata.name", "--no-headers").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return strings.TrimSpace(output)
}
```

### After (type-safe)
```go
func getCurrentLVMClusterName(tc *TestClient) string {
	clusters, err := tc.ListLVMClusters("openshift-lvm-storage")
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(clusters.Items).NotTo(o.BeEmpty(), "No LVMCluster found")

	return clusters.Items[0].Name
}
```

---

## Example 3: getWorkersList (compute_utils.go:150)

### Before
```go
func getWorkersList(oc *exutil.CLI) []string {
	output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("nodes", "-l", "node-role.kubernetes.io/worker", "-o=jsonpath={.items[*].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return strings.Fields(output)
}
```

### After Option A (CLI style)
```go
func getWorkersList(tc *TestClient) []string {
	output, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("nodes", "-l", "node-role.kubernetes.io/worker", "-o=jsonpath={.items[*].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return strings.Fields(output)
}
```

### After Option B (type-safe)
```go
func getWorkersList(tc *TestClient) []string {
	nodeList := &corev1.NodeList{}
	err := tc.List(tc.Context(), nodeList,
		client.MatchingLabels{"node-role.kubernetes.io/worker": ""})
	o.Expect(err).NotTo(o.HaveOccurred())

	var workers []string
	for _, node := range nodeList.Items {
		workers = append(workers, node.Name)
	}
	return workers
}
```

---

## Example 4: waitReady (lvms_utils.go:312)

### Before
```go
func (lvm *lvmCluster) waitReady(oc *exutil.CLI) {
	err := wait.Poll(5*time.Second, 240*time.Second, func() (bool, error) {
		readyFlag, errinfo := lvm.getLvmClusterStatus(oc)
		if errinfo != nil {
			e2e.Logf("Failed to get LvmCluster status: %v, wait for next round to get.", errinfo)
			return false, nil
		}
		if readyFlag == "Ready" {
			e2e.Logf("The LvmCluster \"%s\" have already become Ready to use", lvm.name)
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		lvmClusterDesc := lvm.describeLvmCluster(oc)
		e2e.Logf("oc describe lvmcluster %s:\n%s", lvm.name, lvmClusterDesc)
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("lvmcluster %s not ready", lvm.name))
	}
}
```

### After (type-safe)
```go
func (lvm *lvmCluster) waitReady(tc *TestClient) {
	err := wait.Poll(5*time.Second, 240*time.Second, func() (bool, error) {
		cluster, errinfo := tc.GetLVMCluster(lvm.name, lvm.namespace)
		if errinfo != nil {
			e2e.Logf("Failed to get LvmCluster: %v, wait for next round", errinfo)
			return false, nil
		}
		if cluster.Status.State == lvmv1alpha1.LVMStatusReady {
			e2e.Logf("The LvmCluster \"%s\" is now Ready", lvm.name)
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		// Can still use oc describe for debugging output
		output, _ := tc.AsAdmin().Run("describe").Args("lvmcluster", lvm.name, "-n", lvm.namespace).Output()
		e2e.Logf("oc describe lvmcluster %s:\n%s", lvm.name, output)
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("lvmcluster %s not ready", lvm.name))
	}
}
```

---

## Example 5: pvc.create (persistent_volume_claim_utils.go:108)

### Before
```go
func (pvc *persistentVolumeClaim) create(oc *exutil.CLI) {
	if pvc.namespace == "" {
		pvc.namespace = oc.Namespace()
	}
	err := applyResourceFromTemplate(oc, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}
```

### After Option A (template-based)
```go
func (pvc *persistentVolumeClaim) create(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}
	err := applyResourceFromTemplate(tc, "--ignore-unknown-parameters=true", "-f", pvc.template, "-p", "PVCNAME="+pvc.name, "PVCNAMESPACE="+pvc.namespace, "SCNAME="+pvc.scname,
		"ACCESSMODE="+pvc.accessmode, "VOLUMEMODE="+pvc.volumemode, "PVCCAPACITY="+pvc.capacity)
	o.Expect(err).NotTo(o.HaveOccurred())
}
```

### After Option B (type-safe - no templates)
```go
func (pvc *persistentVolumeClaim) create(tc *TestClient) {
	if pvc.namespace == "" {
		pvc.namespace = tc.Namespace()
	}

	pvcObj := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvc.name,
			Namespace: pvc.namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &pvc.scname,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(pvc.accessmode),
			},
			VolumeMode: (*corev1.PersistentVolumeMode)(&pvc.volumemode),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(pvc.capacity),
				},
			},
		},
	}

	err := tc.Create(tc.Context(), pvcObj)
	o.Expect(err).NotTo(o.HaveOccurred())
}
```

---

## Example 6: execCommandInSpecificNode (compute_utils.go:19)

This function uses `oc debug node` which should remain as CLI command:

### Migration
```go
// Before
func execCommandInSpecificNode(oc *exutil.CLI, nodeHostName string, command string) (output string, err error) {
	debugNodeNamespace := oc.Namespace()
	// ... rest of the function
}

// After - minimal change
func execCommandInSpecificNode(tc *TestClient, nodeHostName string, command string) (output string, err error) {
	debugNodeNamespace := tc.Namespace()
	// ... rest of the function stays the same
}
```

**Note:** `oc debug node` doesn't have a direct API equivalent, so keep using CLI commands for this.

---

## Summary of Migration Strategies

1. **Quick Migration (Phase 1)**
   - Replace `oc *exutil.CLI` → `tc *TestClient`
   - Everything else stays the same
   - Tests work immediately

2. **Type-Safe Migration (Phase 2)**
   - Use `tc.Get()`, `tc.List()`, `tc.Create()` etc.
   - Remove JSON parsing with gjson
   - Direct field access with IDE autocomplete
   - Compiler catches errors

3. **Hybrid Approach (Recommended)**
   - Use type-safe APIs for CRUD operations
   - Keep CLI commands for:
     - `oc debug node`
     - Complex formatting (`-o custom-columns`)
     - Operations without direct API equivalent
     - Template processing (initially)
