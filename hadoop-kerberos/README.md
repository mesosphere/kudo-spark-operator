### How to run

1) Run [deploy.all](deploy-all.sh) script, which will:
 - create namespace
 - install kudo and Spark operator
 - create volumes and configmaps
 - deploy KDC
 - deploy HDFS
 
2) After all the components are in the `Running` state, execute the following: 
```
$ kubectl exec -it <namenode-pod-name> -- init.sh
```
This script will upload test data to HDFS and export a delegation token to `/var/keytabs/` shared folder.

3) Deploy the [app](spark-hdfs-kerberos.yaml): 
```
$ kubectl apply -f spark-hdfs-kerberos.yaml
```
