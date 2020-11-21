# k8s-utils

## kubectl-clean

Find and remove all resources sutisfying selector. If installed in a $PATH can be used as kubectl plugin.

WARNING

`kubectl-clean` is not intended to be used manually. It is deleting resources without asking for a confirmation. When executed with the right parameters it will nuke the cluster! (dry-run option is defauled to true to minimize a risk of accidentaly wiping out kubernetes cluster)

## Usage

```shell
⟫ kubectl clean
Usage of /usr/local/bin/kubectl-clean:
      --annotation-filter string   preserve annotated resources
      --dry-run                    report only (default 'true') (default true)
      --kubeconfig string          kubeconfig file
      --label-selector string      resources to prune (required)
      --namespace string           limit cleanup to a particular namespace
```

## Example

Create some resources and labeled with version=xxx

```shell
⟫ kubectl --namespace playground apply -f deploy1.yaml
deployment.apps/test1 created
⟫ kubectl --namespace playground apply -f deploy2.yaml
deployment.apps/test2 created
⟫ kubectl --namespace playground label deployments.apps test1 version=1
deployment.apps/test1 labeled
⟫ kubectl --namespace playground label deployments.apps test2 version=2
deployment.apps/test2 labeled
```

Find all resources labeled as `version=1`

```shell
⟫ ./kubectl clean --label-selector version=1
2020/07/29 12:55:05 Running GC with: LabelSelector: 'version=1,version', AnnotationFilter: ''
2020/07/29 12:55:07 (dry-run) delete deployments/test1 in namespace playground... OK
```

Find all resources which have label `version` with a value other then `1`

```shell
⟫ ./kubectl clean --label-selector version!=1
2020/07/29 13:00:30 Running GC with: LabelSelector: 'version!=1,version', AnnotationFilter: ''
2020/07/29 13:00:31 (dry-run) delete deployments/test2 in namespace playground... OK
```

Do cleanup keeping only resources labeled with `version=2`

```shell
⟫ kubectl --namespace playground get deployments.apps --show-labels
NAME    READY   UP-TO-DATE   AVAILABLE   AGE   LABELS
test1   1/1     1            1           12m   version=1
test2   1/1     1            1           12m   version=2

⟫ ./kubectl clean --label-selector version!=2 --dry-run=false
2020/07/29 13:04:14 Running GC with: LabelSelector: 'version!=2,version', AnnotationFilter: ''
2020/07/29 13:04:15 delete deployments/test1 in namespace playground... OK

⟫ kubectl --namespace playground get deployments.apps --show-labels
NAME    READY   UP-TO-DATE   AVAILABLE   AGE   LABELS
test2   1/1     1            1           14m   version=2
```
