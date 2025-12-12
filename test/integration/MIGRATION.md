# Migration Guide: From exutil.CLI to TestClient

This guide shows how to migrate from `github.com/openshift/origin/test/extended/util` to our lightweight `TestClient`.

## Benefits

- **Smaller dependencies**: Eliminates the massive `openshift/origin` module
- **Type-safe API**: Direct access to Kubernetes APIs via controller-runtime
- **Better integration**: Uses the same client library as the operator code
- **Faster builds**: Fewer dependencies to download and compile

## Quick Reference

| exutil.CLI | TestClient | Notes |
|------------|------------|-------|
| `exutil.NewCLI("test")` | `NewTestClient("test")` | Creates client |
| `oc.AsAdmin()` | `tc.AsAdmin()` | Admin context |
| `oc.Namespace()` | `tc.Namespace()` | Get namespace |
| `oc.SetupProject()` | `tc.SetupProject()` | Create test namespace |
| `oc.WithoutNamespace()` | `tc.WithoutNamespace()` | No namespace context |
| `oc.AdminKubeClient()` | `tc.AdminKubeClient()` | Get kubernetes clientset |
| `oc.Run("get").Args(...).Output()` | `tc.Run("get").Args(...).Output()` | Execute oc command |
| N/A | `tc.Get(ctx, key, obj)` | Type-safe get (NEW) |
| N/A | `tc.List(ctx, list, opts...)` | Type-safe list (NEW) |
| N/A | `tc.Create(ctx, obj)` | Type-safe create (NEW) |

## Migration Examples

### Example 1: Basic Command Execution

**Before:**
```go
import exutil "github.com/openshift/origin/test/extended/util"

oc := exutil.NewCLI("lvms-test")
output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", "-n", "openshift-lvm-storage", "-o", "json").Output()
```

**After:**
```go
tc := NewTestClient("lvms-test")
output, err := tc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", "-n", "openshift-lvm-storage", "-o", "json").Output()
```

### Example 2: Type-Safe API Access (Recommended)

**Before:**
```go
lvmCluster, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("lvmcluster", "-n", "openshift-lvm-storage", "-o", "json").Output()
lvmClusterState := gjson.Get(lvmCluster, "items.#(metadata.name="+name+").status.state").String()
```

**After:**
```go
lvmCluster, err := tc.GetLVMCluster(name, "openshift-lvm-storage")
if err != nil {
    return "", err
}
lvmClusterState := lvmCluster.Status.State
```

### Example 3: Creating Resources

**Before:**
```go
err := oc.AsAdmin().Run("apply").Args("-f", templatePath).Execute()
```

**After (CLI style):**
```go
err := tc.AsAdmin().Run("apply").Args("-f", templatePath).Execute()
```

**After (Type-safe - NEW):**
```go
lvmCluster := &lvmv1alpha1.LVMCluster{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "test-cluster",
        Namespace: "openshift-lvm-storage",
    },
    Spec: lvmv1alpha1.LVMClusterSpec{
        // ... spec fields
    },
}
err := tc.Create(tc.Context(), lvmCluster)
```

### Example 4: Setup and Teardown

**Before:**
```go
var _ = g.Describe("Test", func() {
    var oc = exutil.NewCLI("lvms-test")

    g.BeforeEach(func() {
        oc.SetupProject()
    })
})
```

**After:**
```go
var _ = g.Describe("Test", func() {
    var tc *TestClient

    g.BeforeEach(func() {
        tc = NewTestClient("lvms-test")
        tc.SetupProject()
    })

    g.AfterEach(func() {
        tc.DeleteNamespace()
    })
})
```

## Step-by-Step Migration Process

### Phase 1: Drop-in Replacement (Easiest)

1. Replace import:
   ```go
   // Remove:
   // import exutil "github.com/openshift/origin/test/extended/util"

   // No new import needed - TestClient is in same package
   ```

2. Replace `exutil.NewCLI()` with `NewTestClient()`:
   ```go
   // Before: oc := exutil.NewCLI("test")
   // After:
   tc := NewTestClient("test")
   ```

3. Replace variable references:
   ```go
   // Before: oc.AsAdmin()...
   // After:  tc.AsAdmin()...
   ```

That's it! Your tests should work with minimal changes.

### Phase 2: Use Type-Safe APIs (Recommended)

After Phase 1 works, gradually migrate to type-safe APIs:

1. Replace JSON parsing with typed structs:
   ```go
   // Before:
   output, _ := oc.Run("get").Args("lvmcluster", name, "-o", "json").Output()
   state := gjson.Get(output, "status.state").String()

   // After:
   lvmCluster, _ := tc.GetLVMCluster(name, namespace)
   state := string(lvmCluster.Status.State)
   ```

2. Use direct API calls instead of `oc` commands:
   ```go
   // Before:
   err := oc.Run("delete").Args("pvc", pvcName, "-n", namespace).Execute()

   // After:
   pvc := &corev1.PersistentVolumeClaim{
       ObjectMeta: metav1.ObjectMeta{
           Name:      pvcName,
           Namespace: namespace,
       },
   }
   err := tc.Delete(tc.Context(), pvc)
   ```

## Common Patterns

### Waiting for Resources

**Before:**
```go
err := wait.Poll(5*time.Second, 240*time.Second, func() (bool, error) {
    output, err := oc.Run("get").Args("pod", podName, "-n", ns, "-o=jsonpath={.status.phase}").Output()
    return output == "Running", err
})
```

**After:**
```go
err := tc.WaitForPodReady(namespace, podName, 240*time.Second)
```

### Getting List of Resources

**Before:**
```go
output, _ := oc.Run("get").Args("pods", "-n", ns, "-o=jsonpath={.items[*].metadata.name}").Output()
podNames := strings.Fields(output)
```

**After:**
```go
podList := &corev1.PodList{}
err := tc.List(tc.Context(), podList, client.InNamespace(ns))
for _, pod := range podList.Items {
    podNames = append(podNames, pod.Name)
}
```

## Need Help?

- The TestClient wraps `sigs.k8s.io/controller-runtime/pkg/client`
- See [controller-runtime docs](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client) for API reference
- Check existing operator code for examples of client usage
