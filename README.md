# namespace-pod-check

The `namespace-pod-check` verifies that pods can be created and deleted across all namespaces. It creates a small BusyBox pod in each namespace, deletes it, and reports success only if every namespace passes.

## Configuration

No additional environment variables are required.

## Build

- `just build` builds the container image locally.
- `just test` runs unit tests.
- `just binary` builds the binary in `bin/`.

## Example HealthCheck

Apply the example below or the provided `healthcheck.yaml`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: namespace-pod-check
  namespace: kuberhealthy
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: namespace-pod-check
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "create", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: namespace-pod-check
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: namespace-pod-check
subjects:
  - kind: ServiceAccount
    name: namespace-pod-check
    namespace: kuberhealthy
---
apiVersion: kuberhealthy.github.io/v2
kind: HealthCheck
metadata:
  name: namespace-pod-check
  namespace: kuberhealthy
spec:
  runInterval: 1h
  timeout: 10m
  podSpec:
    spec:
      serviceAccountName: namespace-pod-check
      containers:
        - name: namespace-pod-check
          image: kuberhealthy/namespace-pod-check:sha-<short-sha>
          imagePullPolicy: IfNotPresent
          resources:
            requests:
              cpu: 15m
              memory: 15Mi
            limits:
              cpu: 25m
      restartPolicy: Always
      terminationGracePeriodSeconds: 5
```
