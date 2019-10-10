The `kudo-spark-operator` is able to seamlessly integrated with Prometheus, which is installed within [Prometheus operator](https://github.com/coreos/prometheus-operator).
Integration with other Prometheus distributions, like kube-prometheus, wasn't tested. 

The `prometheus-operator` uses crafted services discovery approach, introducing `ServiceMonitor` kind. 
But the `kudo-spark-operator` take its configuration burden on itself.

#### How to get `kudo-spark-operator` and app's metrics available for scrape by Prometheus:
1) Ensure `prometheus-operator` is installed on your Kubernetes cluster.
1) Install the KUDO Spark Operator. Metrics reporting is enabled by default and can be disabled by modifying `enableMetrics` parameter.
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
   - Mark your application by the label 
     ```yaml
       labels:
         exposedMetrics: true
     ```

Full blown example you can find in [spark-application.yaml](specs/spark-application.yaml).
