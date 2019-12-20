package tests

import (
	"fmt"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/stretchr/testify/suite"
	"testing"
)

const volcanoInstallerUrl = "https://raw.githubusercontent.com/volcano-sh/volcano/release-0.2/installer/volcano-development.yaml"

type VolcanoIntegrationTestSuite struct {
	operator utils.SparkOperatorInstallation
	suite.Suite
}

func TestVolcanoSuite(t *testing.T) {
	suite.Run(t, new(VolcanoIntegrationTestSuite))
}

func (suite *VolcanoIntegrationTestSuite) SetupSuite() {
	suite.operator = utils.SparkOperatorInstallation{
		Params: map[string]string{
			"enableBatchScheduler": "true",
		},
	}

	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}
	// deploy volcano resources
	_, err := utils.Kubectl("apply", "-f", volcanoInstallerUrl)
	suite.NoError(err)

	// wait until all deployments within a namespace are completed
	utils.Kubectl("wait", "deployment",
		"--all",
		"--for", "condition=available",
		"--namespace", "volcano-system",
		"--timeout=60s")
}

func (suite *VolcanoIntegrationTestSuite) TestAppRunOnVolcano() {
	jobName := "spark-pi"
	sparkPi := utils.SparkJob{
		Name:     jobName,
		Template: fmt.Sprintf("%s.yaml", jobName),
		Params: map[string]interface{}{
			"BatchScheduler": "volcano",
		},
	}
	if err := suite.operator.SubmitJob(&sparkPi); err != nil {
		suite.FailNow(err.Error())
	}

	if err := suite.operator.WaitUntilSucceeded(sparkPi); err != nil {
		suite.FailNow(err.Error())
	}

	// assert that the driver pod was scheduled by volcano
	driverPodName := utils.DriverPodName(jobName)
	component, err := utils.Kubectl("get", "events",
		"--namespace", sparkPi.Namespace,
		"--field-selector", fmt.Sprint("involvedObject.name=", driverPodName),
		"-o", "jsonpath={.items[0].source.component}")
	if suite.NoError(err) {
		suite.Equal("volcano", component)
	}

	// assert that the pod was successfully assigned to a node
	message, err := utils.Kubectl("get", "events",
		"--namespace", sparkPi.Namespace,
		"--field-selector", fmt.Sprint("involvedObject.name=", driverPodName),
		"-o", "jsonpath={.items[0].message}")
	if suite.NoError(err) {
		suite.Contains(message, fmt.Sprintf("Successfully assigned %s/%s", sparkPi.Namespace, driverPodName))
	}
}

func (suite *VolcanoIntegrationTestSuite) TearDownSuite() {
	suite.operator.CleanUp()
	// delete blocks until all resources are deleted
	utils.Kubectl("delete", "-f", volcanoInstallerUrl)
}
