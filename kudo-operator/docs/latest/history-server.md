# Spark History Server Configuration

### Prerequisites

Required software:
* K8s cluster
* [KUDO CLI Plugin](https://kudo.dev/docs/#install-kudo-cli) 0.7.5 or higher

### Installing Spark Operator with History Server Enabled

```
kubectl kudo install ./kudo-operator \
    --namespace <NAMESPACE> \
    -p enableHistoryServer=true \
    -p historyServerFsLogDirectory="s3a://<BUCKET_NAME>/<FOLDER>" \
    -p historyServerOpts="-Dspark.hadoop.fs.s3a.access.key=<AWS_ACCESS_KEY_ID> -Dspark.hadoop.fs.s3a.secret.key=<AWS_SECRET_ACCESS_KEY> -Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem"
```

This will deploy a Pod and Service for the `Spark History Server` with the `Spark Event Log` directory configured via the `historyServerFsLogDirectory` parameter. This is an S3 backed storage for event logs. Spark Operator also supports Persistent Volume Claim (PVC) based storage. There is a parameter `historyServerPVCName` to pass the name of the PVC. Make sure that provided PVC should have `ReadWriteMany` [access mode](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes) supported.

### Creating Spark Application

Make sure `specs/spark-application.yaml` has these properties specified under `spec.sparkConf`:

```
"spark.eventLog.enabled": "true"
"spark.eventLog.dir": "s3a://<BUCKET_NAME>/<FOLDER>"
"spark.hadoop.fs.s3a.access.key": "<AWS_ACCESS_KEY_ID>"
"spark.hadoop.fs.s3a.secret.key": "<AWS_SECRET_ACCESS_KEY>"
```

If PVC is passed while installing the Spark Operator, make sure these two fields are also added with following values:

```
# Add this under SparkApplicationSpec
Volumes:
- name: pvc-storage
  persistentVolumeClaim:
    claimName: <HISTORY_SERVER_PVC_NAME>

# Add this under SparkPodSpec
volumeMounts:
- mountPath: <HISTORY_SERVER_FS_LOG_DIRECTORY>
  name: pvc-storage
```

Make sure that CRD and RBAC for SparkApplication are already created. Now we can run the application as follows:

```
kubectl apply -n <NAMESPACE> -f specs/spark-application.yaml
```

### Accessing Spark History Server UI

 **Using Port-Forwarding:**

You can run this command to expose the UI in your local machine.

```
kubectl port-forward <HISTORY_SERVER_POD_NAME> 18080:18080
```

Verify local access using this:

```
curl -L localhost:18080
```

**Using LoadBalancer:**

Create a Service with type as `LoadBalancer` which will expose the Spark History Server UI. Service specification will be as follows:

```
apiVersion: v1
kind: Service
metadata:
  labels:
    app: history-server-ui-lb
  name: history-server-ui-lb
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: history-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 18080
```

Create service with following command:

```
kubectl create -f history-server-svc.yaml -n spark
```

Wait for few minutes and verify the access to Spark History Server UI via external address as follows:

```
curl -L http://$(kubectl get svc history-server-ui-lb -n spark --output jsonpath='{.status.loadBalancer.ingress[*].hostname}')
```

### List of available parameters for Spark History Server

| Parameter Name               | Default Value |  Description                                                                                                              |
| --------------               | ------------- |  -----------                                                                                                              |
| enableHistoryServer          | false         | Enable Spark History Server                                                                                               |
| historyServerFsLogDirectory  | ""            | Spark EventLog Directory from which to load events for prior Spark job runs (e.g., hdfs://hdfs/ or s3a://path/to/bucket). |
| historyServerCleanerEnabled  | false         | Specifies whether the Spark History Server should periodically clean up event logs from storage.                          |
| historyServerCleanerInterval | 1d            | Frequency the Spark History Server checks for files to delete.                                                            |
| historyServerCleanerMaxAge   | 7d            | History files older than this will be deleted.                                                                            |
| historyServerOpts            | ""            | Extra options to pass to the Spark History Server                                                                         |
| historyServerPVCName         | ""            | External Persistent Volume Claim Name used for Spark event logs storage                                                   |

Note: Value passed as parameter will get the priority over the value passed as option in the parameter `historyServerOpts`.
