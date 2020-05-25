package tests

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"io"
	v12 "k8s.io/api/core/v1"
	. "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

const dashboardsDir = "../operators/repository/spark/docs/latest/resources/dashboards"
const jobName = "mock-task-runner"
const jobTemplate = "spark-mock-task-runner-with-monitoring.yaml"
const prometheusNamespace = "kubeaddons"
const prometheusPort = 9090
const queryTimeout = 1 * time.Minute
const queryRetryDelay = 5 * time.Second
const contextTimeout = 10 * time.Second

type MetricsTestSuite struct {
	operator utils.SparkOperatorInstallation
	suite.Suite
}

type PortForwardProps struct {
	Pod         v12.Pod
	LocalPort   int
	PodPort     int
	Out, ErrOut io.Writer
	// StopCh channel is used to stop portforward
	StopCh <-chan struct{}
	// ReadyCh channel is updated when portforward is ready
	ReadyCh chan struct{}
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}

func (suite *MetricsTestSuite) SetupSuite() {
	suite.operator = utils.SparkOperatorInstallation{
		Namespace: "spark-operator-metrics",
		Params: map[string]string{
			"enableMetrics": "true",
		},
	}

	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow("Error while installing the operator", err)
	}
}

func (suite *MetricsTestSuite) TearDownSuite() {
	suite.operator.CleanUp()
}

func (suite *MetricsTestSuite) TestMetricsInPrometheus() {
	// capture test start time for later use in Prometheus queries
	testStartTime := time.Now()

	// to initiate application-specific metrics generation, we need to create a workload by submitting
	// two applications with different results (successful and failed).
	if err := submitJobs(suite); err != nil {
		suite.FailNow("Error while submitting a job", err)
	}

	// get prometheus pod name
	prometheusPodName, err := utils.Kubectl("get", "pod",
		"--namespace", prometheusNamespace,
		"--selector", "app=prometheus",
		"--output", "jsonpath={.items[*].metadata.name}")

	if err != nil {
		suite.FailNow("Prometheus pod not found", err)
	}

	// start a port-forward as a go-routine to directly communicate with Prometheus API
	stopCh, readyCh := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	go func() {
		err := startPortForward(PortForwardProps{
			Pod: v12.Pod{
				ObjectMeta: ObjectMeta{
					Name:      prometheusPodName,
					Namespace: prometheusNamespace,
				},
			},
			LocalPort: prometheusPort,
			PodPort:   prometheusPort,
			Out:       out,
			ErrOut:    errOut,
			StopCh:    stopCh,
			ReadyCh:   readyCh,
		})
		if err != nil {
			suite.FailNow("Error while creating port-forward", err)
		}
	}()

	select {
	case <-readyCh:
		if len(errOut.String()) != 0 {
			suite.FailNow(errOut.String())
		} else if len(out.String()) != 0 {
			log.Info(out.String())
		}
		break
	}

	client, err := api.NewClient(api.Config{
		Address: fmt.Sprintf("http://localhost:%d", prometheusPort),
	})
	if err != nil {
		suite.Fail(err.Error(), "Error creating Prometheus client")
	}

	operatorPodName, err := suite.operator.GetOperatorPodName()
	if err != nil {
		suite.FailNow("Error getting operator pod name", err)
	}

	// collect and prepare Prometheus queries
	queries, err := collectQueries([]string{
		"$Spark_Operator_Instance", operatorPodName,
		"$app_name", jobName,
		"$namespace", suite.operator.Namespace,
		"\\\"", "\""})

	if err != nil {
		suite.FailNow("Error parsing Prometheus queries", err)
	}

	v1api := v1.NewAPI(client)

	timeRange := v1.Range{
		Start: testStartTime,
		End:   time.Now().Add(10 * time.Minute),
		Step:  10 * time.Second,
	}
	for _, query := range queries {
		if err := suite.queryPrometheus(query, v1api, timeRange); err != nil {
			suite.Failf("Error while executing the query \"%s\"", query, err)
		}
	}
	// stop PortForward connection
	close(stopCh)
}

func (suite *MetricsTestSuite) queryPrometheus(query string, v1api v1.API, timeRange v1.Range) error {
	return utils.RetryWithTimeout(queryTimeout, queryRetryDelay, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
		defer cancel()
		log.Infof("Executing query: \"%s\"", query)
		result, warnings, err := v1api.QueryRange(ctx, query, timeRange)
		if err != nil {
			log.Errorf(": %v", err)
			return err
		}
		if len(warnings) > 0 {
			log.Warnf("Warnings: %v\n", warnings)
		}
		if len(result.String()) == 0 {
			return fmt.Errorf("no metrics found for query %v", query)
		}
		log.Infof("Result: %v", result)
		return nil
	})
}

// submitJobs creates two SparkApplications: the first one completes successfully, the second one with mis-configured
// arguments and should fail. With this approach an operator will generate all the required metrics that used in queries
// under test
func submitJobs(suite *MetricsTestSuite) error {
	job := utils.SparkJob{
		Name:     fmt.Sprintf("%s-failed", jobName),
		Template: jobTemplate,
		Params: map[string]interface{}{
			"args": []string{"1"},
		},
	}

	if err := suite.operator.SubmitJob(&job); err != nil {
		return err
	}

	job = utils.SparkJob{
		Name:     jobName,
		Template: jobTemplate,
		Params: map[string]interface{}{
			"args": []string{"2", "120"},
		},
	}

	if err := suite.operator.SubmitJob(&job); err != nil {
		return err
	}

	if err := suite.operator.WaitUntilSucceeded(job); err != nil {
		return err
	}
	return nil
}

// this method 'grep's all prometheus queries from dashboards files located in 'dashboardsDir'
// and replaces metric label placeholders with the real data
func collectQueries(replacements []string) ([]string, error) {

	// define metrics which cannot be verified
	var excludedMetrics = []string{
		"spark_app_executor_failure_count",
	}

	command := exec.Command("grep", "--no-filename", "\"expr\"",
		fmt.Sprintf("%s/%s", dashboardsDir, "grafana_spark_operator.json"),
		fmt.Sprintf("%s/%s", dashboardsDir, "grafana_spark_applications.json"))

	output, err := command.CombinedOutput()
	if err != nil {
		log.Error(string(output))
		return nil, err
	}
	replacer := strings.NewReplacer(replacements...)

	var queries []string
	pattern := regexp.MustCompile("\"expr\": \"([a-z_\\d(]+{.*}\\)?)")

	for _, line := range strings.Split(string(output), "\n") {
		if len(line) > 0 {
			query := pattern.FindStringSubmatch(strings.TrimSpace(line))[1]
			query = replacer.Replace(query)
			for _, metric := range excludedMetrics {
				if !strings.Contains(query, metric) {
					queries = append(queries, query)
				}
			}
		}
	}
	return queries, nil
}

// this method creates port forwarding request based on 'PortForwardProps'
func startPortForward(props PortForwardProps) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", props.Pod.Namespace, props.Pod.Name)
	config, err := clientcmd.BuildConfigFromFlags("", utils.KubeConfig)

	if err != nil {
		return err
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(config)

	if err != nil {
		return err
	}

	serverURL := url.URL{Scheme: "https", Path: path, Host: strings.TrimLeft(config.Host, "htps:/")}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", props.LocalPort, props.PodPort)},
		props.StopCh, props.ReadyCh, props.Out, props.ErrOut)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}
