package tests

import (
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/fatih/structs"
	"github.com/iancoleman/strcase"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "k8s.io/client-go/tools/leaderelection/resourcelock"
	"strconv"
	"strings"
	"testing"
	"time"
)

const electionRecordRetryInterval = 2 * time.Second
const electionRecordRetryTimeout = 30 * time.Second
const processingKeyLogRecordFormat = "Starting processing key: \"%s/%s\""

type LeaderElectionParameters struct {
	Replicas                    int
	EnableLeaderElection        bool
	LeaderElectionLockName      string
	LeaderElectionLeaseDuration string
	LeaderElectionRenewDeadline string
	LeaderElectionRetryPeriod   string
}

type HighAvailabilityTestSuite struct {
	leaderElectionParams LeaderElectionParameters
	operator             utils.SparkOperatorInstallation
	suite.Suite
}

func TestHASuite(t *testing.T) {
	suite.Run(t, new(HighAvailabilityTestSuite))
}

func (suite *HighAvailabilityTestSuite) SetupSuite() {
	suite.leaderElectionParams = LeaderElectionParameters{
		Replicas:                    3,
		EnableLeaderElection:        true,
		LeaderElectionLockName:      "leader-election-lock",
		LeaderElectionLeaseDuration: "15s",
		LeaderElectionRenewDeadline: "10s",
		LeaderElectionRetryPeriod:   "3s",
	}
	if paramsMap, err := convertStructToMap(suite.leaderElectionParams); err != nil {
		suite.FailNow(err.Error())
	} else {
		suite.operator = utils.SparkOperatorInstallation{
			Params: paramsMap,
		}
	}

	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}
	utils.Kubectl("wait", "deployment", "--all", "--for", "condition=available",
		"--namespace", suite.operator.Namespace, "--timeout=60s")
}

func (suite *HighAvailabilityTestSuite) TearDownSuite() {
	suite.operator.CleanUp()
}

func (suite *HighAvailabilityTestSuite) Test_LeaderElectionConfiguration() {
	operator := suite.operator
	params := suite.leaderElectionParams
	args, err := utils.Kubectl("get", "deployment", operator.InstanceName, "-n", operator.Namespace,
		"-o=jsonpath={.spec.template.spec.containers[0].args}")
	if err != nil {
		suite.FailNow(err.Error())
	}
	availableReplicas, _ := utils.Kubectl("get", "deployment", operator.InstanceName,
		"-n", operator.Namespace,
		"-o=jsonpath={.status.availableReplicas}")

	suite.Equal(strconv.Itoa(params.Replicas), availableReplicas)
	suite.Contains(args, fmt.Sprint("-leader-election=", params.EnableLeaderElection))
	suite.Contains(args, fmt.Sprint("-leader-election-lock-name=", params.LeaderElectionLockName))
	suite.Contains(args, fmt.Sprint("-leader-election-lock-namespace=", operator.Namespace))
	suite.Contains(args, fmt.Sprint("-leader-election-lease-duration=", params.LeaderElectionLeaseDuration))
	suite.Contains(args, fmt.Sprint("-leader-election-renew-deadline=", params.LeaderElectionRenewDeadline))
	suite.Contains(args, fmt.Sprint("-leader-election-retry-period=", params.LeaderElectionRetryPeriod))
}

func (suite *HighAvailabilityTestSuite) Test_LeaderElectionRecord() {
	leaderElectionRecord, err := getLeaderElectionRecord(suite.operator)
	if suite.NoError(err) {
		suite.NotEmpty(leaderElectionRecord.HolderIdentity)
	}
}

func (suite *HighAvailabilityTestSuite) Test_LeaderFailover() {
	// print the current deployment state
	utils.Kubectl("describe", "deployment", suite.operator.InstanceName, "-n", suite.operator.Namespace)
	utils.Kubectl("get", "all", "-n", suite.operator.Namespace)

	operator := suite.operator
	leaderElectionRecord, err := getLeaderElectionRecord(operator)
	if err != nil {
		suite.FailNow(err.Error())
	}

	fmt.Println("Current leader: ", leaderElectionRecord.HolderIdentity)
	// deploy workload
	jobName := "mock-task-runner"
	mockTaskRunner := utils.SparkJob{
		Name:     jobName,
		Template: "spark-mock-task-runner-job.yaml",
		Params: map[string]interface{}{
			"args": []string{"1", "30"},
		},
	}
	if err := operator.SubmitJob(&mockTaskRunner); err != nil {
		suite.FailNow(err.Error())
	}
	// wait until the application is running
	if err := operator.WaitForJobState(mockTaskRunner, v1beta2.RunningState); err != nil {
		suite.FailNow(err.Error())
	}

	// check leader started processing the application
	logContains, err := utils.PodLogContains(mockTaskRunner.Namespace, leaderElectionRecord.HolderIdentity,
		fmt.Sprintf(processingKeyLogRecordFormat, mockTaskRunner.Namespace, mockTaskRunner.Name))
	if suite.NoError(err) {
		suite.True(logContains)
	}

	log.Infof("deleting current leader pod \"%s\"", leaderElectionRecord.HolderIdentity)
	if err := utils.DeleteResource(operator.Namespace, "pod", leaderElectionRecord.HolderIdentity); err != nil {
		suite.FailNow(err.Error())
	}
	var newLeaderPodName string
	// check re-election
	if err := utils.RetryWithTimeout(electionRecordRetryTimeout, electionRecordRetryInterval, func() error {
		if newLeaderElectionRecord, err := getLeaderElectionRecord(operator); err != nil {
			return err
		} else if newLeaderElectionRecord.HolderIdentity == leaderElectionRecord.HolderIdentity {
			return errors.New("Waiting for the new leader to be elected")
		} else {
			log.Info("New leader found: ", newLeaderElectionRecord.HolderIdentity)
			newLeaderPodName = newLeaderElectionRecord.HolderIdentity
		}
		return nil
	}); err != nil {
		suite.FailNow(err.Error())
	}

	suite.NoError(operator.WaitForJobState(mockTaskRunner, v1beta2.CompletedState))

	// check the new leader started processing the application
	logContains, err = utils.PodLogContains(mockTaskRunner.Namespace, newLeaderPodName,
		fmt.Sprintf(processingKeyLogRecordFormat, mockTaskRunner.Namespace, mockTaskRunner.Name))

	if suite.NoError(err) {
		suite.True(logContains)
	}
}

func getLeaderElectionRecord(operator utils.SparkOperatorInstallation) (*LeaderElectionRecord, error) {
	lockName := operator.Params["leaderElectionLockName"]
	var leaderElectionRecord *LeaderElectionRecord
	err := utils.RetryWithTimeout(electionRecordRetryTimeout, electionRecordRetryInterval, func() error {
		configMap, err := operator.K8sClients.CoreV1().ConfigMaps(operator.Namespace).Get(lockName, v1.GetOptions{})
		if err != nil {
			return err
		} else if configMap == nil {
			return errors.New("LeaderElectionRecord hasn't been created.")
		}
		leaderElectionRecordString := configMap.GetAnnotations()[LeaderElectionRecordAnnotationKey]
		if err := json.Unmarshal([]byte(leaderElectionRecordString), &leaderElectionRecord); err != nil {
			return err
		}
		if len(strings.TrimSpace(leaderElectionRecord.HolderIdentity)) == 0 {
			return errors.New("No leader is currently elected.")
		} else {
			log.Info("LeaderElectionRecord: ", *leaderElectionRecord)
			// check, that leader pod exists
			if _, err := operator.K8sClients.CoreV1().Pods(operator.Namespace).Get(leaderElectionRecord.HolderIdentity, v1.GetOptions{}); err != nil {
				return err
			}
		}
		return nil
	})
	return leaderElectionRecord, err
}

func convertStructToMap(params interface{}) (map[string]string, error) {
	paramsMap := make(map[string]string)
	fields := structs.Fields(params)
	for _, field := range fields {
		key := strcase.ToLowerCamel(field.Name())
		switch v := field.Value().(type) {
		default:
			return paramsMap, fmt.Errorf("unexpected type %T", v)
		case int:
			paramsMap[key] = strconv.Itoa(field.Value().(int))
		case string:
			paramsMap[key] = field.Value().(string)
		case bool:
			paramsMap[key] = strconv.FormatBool(field.Value().(bool))
		}
	}
	return paramsMap, nil
}
