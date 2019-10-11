Out of the box, the `kudo-spark-operator` has enabled metrics reporting. 
By default, it supports integration with the [Prometheus operator](https://github.com/coreos/prometheus-operator).

The `prometheus-operator` uses crafted services discovery approach, introducing `ServiceMonitor` kind. 
Prometheus Operator relies on `ServiceMonitor` kind which describes the set of targets to be monitored. 
KUDO Spark Operator configures `ServiceMonitor`s for both the Operator and submitted Spark Applications automatically 
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
   - use Spark image with JMXPrometheus exporter jar on the board i.e. `mesosphere/spark:2.4.4-bin-hadoop2.7-k8s` 
   - enable driver/executors monitoring by adding the yaml piece into `spec` section:
     ```yaml
       monitoring:
         exposeDriverMetrics: true
         exposeExecutorMetrics: true
         prometheus:
           jmxExporterJar: "/prometheus/jmx_prometheus_javaagent-0.11.0.jar"
           port: 8090
     ```  
   - if it's necessary to expose metrics endpoint on other than 8090 port, then, please, 
     1) change the port value in `SparkApplication` definition yaml (`spec.monitoring.prometheus.port`) and 
     1) specify appropriate parameter during `kudo-spark-operator` installation - `-p appMetricsPort=<desired_port>`. 
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
