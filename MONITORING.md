The `kudo-spark-operator` is able to seamlessly integrated with Prometheus, which is installed within [Prometheus operator](https://github.com/coreos/prometheus-operator).
Integration with other Prometheus distributions, like kube-prometheus, wasn't tested. 

The `prometheus-operator` uses crafted services discovery approach, introducing `ServiceMonitor` kind. 
Prometheus Operator relies on ServiceMonitor kind which describes the set of targets to be monitored. 
KUDO Spark Operator configures ServiceMonitors for both the Operator and submitted Spark Applications automatically 
when monitoring is enabled.

#### Configuring Spark Operator and Spark Application metrics export to Prometheus
1) Ensure `prometheus-operator` is installed on your Kubernetes cluster.
1) Install the KUDO Spark Operator. Metrics reporting is enabled by default and can be disabled by modifying `enableMetrics` parameter.
1) Create ServiceMonitor for Spark (see prometheus-operator docs). Take this yaml without modification - 
   ```yaml
   cat <<EOF | kubectl apply -f -
   apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
   metadata:
     labels:
       app: prometheus-operator
       release: prometheus-kubeaddons
     name: spark-cluster-monitor
   spec:
     endpoints:
       - interval: 5s
         port: metrics
     selector:
       matchLabels:
         spark/servicemonitor: "true"
   EOF
   ```
1) Composing your Spark Application yaml:
   - use Spark image with JMXPrometheus exporter jar on the board i.e. `gcr.io/spark-operator/spark:v2.4.4-gcs-prometheus` 
   - enable driver/executors monitoring by adding the yaml piece into `spec` section:
     ```yaml
       monitoring:
         exposeDriverMetrics: true
         exposeExecutorMetrics: true
         prometheus:
           jmxExporterJar: "/prometheus/jmx_prometheus_javaagent-0.11.0.jar"
           port: 8090
     ```  
   - if you would like use other than 8090 port for metrics exporting, you must pass appropriate parameter during `kudo-spark-operator` installation `-p appMetricsPort=<desired_port>` 
   - Mark `driver` and/or `executor` with the label `metrics-exposed: "true"` -
     ```yaml
     spec:
       driver:
         labels:
            metrics-exposed: "true"
       executor:
         labels:
           metrics-exposed: "true"
     ```

Full configuration example is available in [specs/spark-application.yaml](specs/spark-application.yaml).
