# Expected Output

When you run this example, here's what you should see:

## 1. After Applying ListSource

```bash
$ kubectl apply -f listsource.yaml
listsource.batchops.example.com/basic-list created

$ kubectl get listsource
NAME          TYPE     ITEMS   AGE
basic-list    static   3       5s

$ kubectl get configmap basic-list-data
NAME              DATA   AGE
basic-list-data   3      5s
```

## 2. ConfigMap Contents

```bash
$ kubectl get configmap basic-list-data -o yaml
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: basic-list-data
  ownerReferences:
    - apiVersion: batchops.example.com/v1alpha1
      kind: ListSource
      name: basic-list
data:
  ITEM_0: alice
  ITEM_1: bob
  ITEM_2: charlie
```

## 3. After Applying ListJob

```bash
$ kubectl apply -f listjob.yaml
listjob.batchops.example.com/basic-processor created

$ kubectl get listjob
NAME              ACTIVE   SUCCEEDED   FAILED   AGE
basic-processor   2        0           0        2s
```

## 4. Jobs Being Created

```bash
$ kubectl get jobs
NAME                COMPLETIONS   DURATION   AGE
basic-processor-0   0/1           2s         2s
basic-processor-1   0/1           2s         2s
basic-processor-2   0/1           1s         1s
```

Notice that only 2 jobs run in parallel initially (as configured by `parallelism: 2`), and the third job starts after one completes.

## 5. Job Pods Running

```bash
$ kubectl get pods
NAME                      READY   STATUS      RESTARTS   AGE
basic-processor-0-xxxxx   1/1     Running     0          3s
basic-processor-1-xxxxx   1/1     Running     0          3s
```

## 6. Job Logs

```bash
$ kubectl logs basic-processor-0-xxxxx
Processing item: alice
Item processed successfully!

$ kubectl logs basic-processor-1-xxxxx
Processing item: bob
Item processed successfully!

$ kubectl logs basic-processor-2-xxxxx
Processing item: charlie
Item processed successfully!
```

## 7. Final Status (All Jobs Completed)

```bash
$ kubectl get jobs
NAME                COMPLETIONS   DURATION   AGE
basic-processor-0   1/1           3s         15s
basic-processor-1   1/1           3s         15s
basic-processor-2   1/1           3s         14s

$ kubectl get listjob
NAME              ACTIVE   SUCCEEDED   FAILED   AGE
basic-processor   0        3           0        15s
```

## 8. ListJob Status Details

```bash
$ kubectl get listjob basic-processor -o jsonpath='{.status}' | jq
```

```json
{
  "active": 0,
  "succeeded": 3,
  "failed": 0,
  "conditions": [
    {
      "type": "Complete",
      "status": "True",
      "lastTransitionTime": "2025-02-03T10:15:30Z",
      "reason": "AllJobsSucceeded",
      "message": "All 3 jobs completed successfully"
    }
  ]
}
```

## Troubleshooting

### No Jobs Created

If jobs aren't being created:

1. Check ListSource status:
   ```bash
   kubectl describe listsource basic-list
   ```

2. Check that ConfigMap was created:
   ```bash
   kubectl get configmap basic-list-data
   ```

3. Check Parallax operator logs:
   ```bash
   kubectl logs -n parallax-system deployment/parallax -f
   ```

### Jobs Failing

If jobs are failing:

1. Check pod logs:
   ```bash
   kubectl logs basic-processor-0-xxxxx
   ```

2. Describe the failed pod:
   ```bash
   kubectl describe pod basic-processor-0-xxxxx
   ```

3. Check job events:
   ```bash
   kubectl describe job basic-processor-0
   ```
