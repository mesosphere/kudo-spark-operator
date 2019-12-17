package tests

import (
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "k8s.io/client-go/tools/leaderelection/resourcelock"
	"strings"
	"testing"
	"time"
)

const electionRecordRetryInterval = 2 * time.Second
const electionRecordRetryTimeout = 30 * time.Second

type HighAvailabilityTestSuite struct {
	leaderElectionParams map[string]string
	operator             utils.SparkOperatorInstallation
	suite.Suite
}

func TestHASuite(t *testing.T) {
	suite.Run(t, new(HighAvailabilityTestSuite))
}

func (suite *HighAvailabilityTestSuite) SetupSuite() {
	suite.leaderElectionParams = map[string]string{
		"replicas":                    "3",
		"enableLeaderElection":        "true",
		"leaderElectionLockName":      "lock",
		"leaderElectionLeaseDuration": "15s",
		"leaderElectionRenewDeadline": "10s",
		"leaderElectionRetryPeriod":   "3s",
	}
	suite.operator = utils.SparkOperatorInstallation{
		Params: suite.leaderElectionParams,
	}

	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}

}

func (suite *HighAvailabilityTestSuite) TearDownSuite() {
	suite.operator.CleanUp()
}

func (suite *HighAvailabilityTestSuite) TestParameters() {
	operator := suite.operator
	params := suite.leaderElectionParams
	args, err := utils.Kubectl("get", "deployment", operator.InstanceName, "-n", operator.Namespace,
		"-o=jsonpath={.spec.template.spec.containers[0].args}")
	if err != nil {
		suite.FailNow(err.Error())
	}
	availableReplicas, _ := utils.Kubectl("get", "deployment", operator.InstanceName, "-n", operator.Namespace,
		"-o=jsonpath={.status.availableReplicas}")

	suite.Equal(params["replicas"], availableReplicas)
	suite.Contains(args, fmt.Sprint("-leader-election=", params["enableLeaderElection"]))
	suite.Contains(args, fmt.Sprint("-leader-election-lock-name=", params["leaderElectionLockName"]))
	suite.Contains(args, fmt.Sprint("-leader-election-lock-namespace=", operator.Namespace))
	suite.Contains(args, fmt.Sprint("-leader-election-lease-duration=", params["leaderElectionLeaseDuration"]))
	suite.Contains(args, fmt.Sprint("-leader-election-renew-deadline=", params["leaderElectionRenewDeadline"]))
	suite.Contains(args, fmt.Sprint("-leader-election-retry-period=", params["leaderElectionRetryPeriod"]))
}

func (suite *HighAvailabilityTestSuite) TestLeaderElection() {
	leaderElectionRecord, err := getLeaderElectionRecord(suite.operator)
	if suite.NoError(err) {
		suite.NotEmpty(leaderElectionRecord.HolderIdentity)
	}
}

func (suite *HighAvailabilityTestSuite) TestFailover() {
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

	log.Infof("deleting current leader pod \"%s\"", leaderElectionRecord.HolderIdentity)
	err = utils.DeleteResource(operator.Namespace, "pod", leaderElectionRecord.HolderIdentity)

	// check re-election
	if err := utils.RetryWithTimeout(electionRecordRetryTimeout, electionRecordRetryInterval, func() error {
		if newLeaderElectionRecord, err := getLeaderElectionRecord(operator); err != nil {
			return err
		} else if newLeaderElectionRecord.HolderIdentity == leaderElectionRecord.HolderIdentity {
			return errors.New("Waiting for the new leader to be elected")
		} else {
			log.Info("New leader found: ", newLeaderElectionRecord.HolderIdentity)
		}
		return nil
	}); err != nil {
		suite.FailNow(err.Error())
	}

	suite.NoError(operator.WaitForJobState(mockTaskRunner, v1beta2.CompletedState))
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
		}
		return nil
	})
	return leaderElectionRecord, err
}
