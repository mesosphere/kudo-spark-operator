package tests

import (
	"fmt"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/mesosphere/kudo-spark-operator/tests/utils"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var counter int

type SparkEncryptionSuite struct {
	operator utils.SparkOperatorInstallation
	// name of Secret object with sensitive data
	sparkSecrets string
	// secret used for RPC authentication
	authSecret string
	// password for private key
	keyPassword string
	// password for keystore
	keyStorePassword string
	// password for truststore
	trustStorePassword string
	keyStorePath       string
	trustStorePath     string
	// Spark config properties
	sparkConf map[string]string
	// name of a Secret with key-stores
	sslSecrets string
	suite.Suite
}

func TestSparkEncryptionSuite(t *testing.T) {
	testSuite := SparkEncryptionSuite{
		sparkSecrets:       "secrets",
		authSecret:         "changeit",
		keyPassword:        "changeit",
		keyStorePassword:   "changeit",
		trustStorePassword: "changeit",
		sslSecrets:         "ssl-secrets",
		keyStorePath:       "/tmp/spark/ssl/keystore.jks",
		trustStorePath:     "/tmp/spark/ssl/truststore.jks",
	}
	suite.Run(t, &testSuite)
}

func (suite *SparkEncryptionSuite) SetupSuite() {
	suite.operator = utils.SparkOperatorInstallation{}
	if err := suite.operator.InstallSparkOperator(); err != nil {
		suite.FailNow(err.Error())
	}
	suite.prepareKeyStores()
	suite.createSecrets()
}

func (suite *SparkEncryptionSuite) createSecrets() {
	sparkSecrets := map[string][]byte{
		"auth-secret":         []byte(suite.authSecret),
		"key-password":        []byte(suite.keyPassword),
		"keystore-password":   []byte(suite.keyStorePassword),
		"truststore-password": []byte(suite.trustStorePassword),
	}

	if err := utils.CreateSecretEncoded(
		suite.operator.K8sClients,
		suite.sparkSecrets,
		suite.operator.Namespace,
		sparkSecrets); err != nil {
		suite.FailNowf("Error while creating secret \"%s\"", suite.sparkSecrets, err)
	}

	keystore, _ := ioutil.ReadFile(suite.keyStorePath)
	truststore, _ := ioutil.ReadFile(suite.trustStorePath)

	if err := utils.CreateSecretEncoded(suite.operator.K8sClients,
		suite.sslSecrets,
		suite.operator.Namespace,
		map[string][]byte{
			"keystore.jks":   keystore,
			"truststore.jks": truststore,
		}); err != nil {
		suite.FailNowf("Error while creating secret \"%s\"", suite.sslSecrets, err)
	}
}

func (suite *SparkEncryptionSuite) TearDownSuite() {
	suite.operator.CleanUp()
}

func (suite *SparkEncryptionSuite) TestRpc() {
	sparkConf := map[string]string{
		"spark.authenticate": "true",
	}
	suite.Run("TestAuth", func() {
		assertSparkApp(suite, sparkConf, []string{"1", "1"})
	})

	sparkConf["spark.network.crypto.enabled"] = "true"
	suite.Run("TestNetworkEncryption", func() {
		assertSparkApp(suite, sparkConf, []string{"1", "1"})
	})

	sparkConf["spark.authenticate.enableSaslEncryption"] = "true"
	suite.Run("TestSaslEncryption", func() {
		assertSparkApp(suite, sparkConf, []string{"1", "1"})
	})
}

func (suite *SparkEncryptionSuite) TestSSL() {
	sparkConf := map[string]string{
		"spark.ssl.enabled":    "true",
		"spark.ssl.keyStore":   suite.keyStorePath,
		"spark.ssl.protocol":   "TLSv1.2",
		"spark.ssl.trustStore": suite.trustStorePath,
	}
	assertSparkApp(suite, sparkConf, []string{"1", "20"})
}

// method creates required key stores for Spark SSL configuration
func (suite *SparkEncryptionSuite) prepareKeyStores() {
	const commandName = "keytool"
	const alias = "selfsigned"
	const tempDir = "/tmp/spark/ssl"
	certPath := fmt.Sprint(tempDir, "/", "test.cert")

	if err := exec.Command("mkdir", "-p", tempDir).Run(); err != nil {
		suite.FailNowf("Can't create a temp dir \"%s\"", tempDir, err)
	}
	// generate a public-private key pair
	genKeyPairCmd := []string{
		"-genkeypair",
		"-keystore", suite.keyStorePath,
		"-keyalg", "RSA",
		"-alias", alias,
		"-dname", "CN=sparkcert OU=KUDO O=D2IQ L=SF S=CA C=US",
		"-storepass", suite.keyStorePassword,
		"-keypass", suite.keyPassword,
	}
	// export the generated certificate
	exportCertCmd := []string{
		"-exportcert",
		"-keystore", suite.keyStorePath,
		"-alias", alias,
		"-storepass", suite.keyStorePassword,
		"-file", certPath,
	}
	// import the certificate into a truststore
	importCertCmd := []string{
		"-importcert",
		"-keystore", suite.trustStorePath,
		"-alias", alias,
		"-storepass", suite.trustStorePath,
		"-file", certPath,
		"-noprompt",
	}

	prepKeyStoresCommandChain := [][]string{genKeyPairCmd, exportCertCmd, importCertCmd}

	for _, commandArgs := range prepKeyStoresCommandChain {
		_, err := utils.RunAndLogCommandOutput(exec.Command(commandName, commandArgs...))
		if err != nil {
			suite.FailNow("Error while preparing the key-stores", err)
		}
	}

	suite.Assert().FileExists(suite.keyStorePath)
	suite.Assert().FileExists(suite.trustStorePath)
	suite.Assert().FileExists(certPath)
}

// launches a Spark application based on `sparkConf` and asserts its successful completion
func assertSparkApp(suite *SparkEncryptionSuite, sparkConf map[string]string, args []string) {
	counter++
	appName := fmt.Sprintf("spark-mock-task-runner-encrypted-%d", counter)

	_, authEnabled := sparkConf["spark.authenticate"]
	_, sslEnabled := sparkConf["spark.ssl.enabled"]

	sparkApp := utils.SparkJob{
		Name:     appName,
		Template: "spark-mock-task-runner-encrypted.yaml",
		Params: map[string]interface{}{
			"Args":         args,
			"SparkConf":    sparkConf,
			"SparkSecrets": suite.sparkSecrets,
			"SslSecrets":   suite.sslSecrets,
			"AuthEnabled":  authEnabled,
			"SslEnabled":   sslEnabled,
		},
	}

	suite.Assert().NoError(suite.operator.SubmitJob(&sparkApp))
	defer suite.operator.DeleteJob(sparkApp)

	// when ssl is configured, check Spark UI is accessible via https on 4440 port
	if sslEnabled {
		checkSparkUI(appName, sparkApp, suite)
	}

	suite.Assert().NoError(suite.operator.WaitUntilSucceeded(sparkApp))
}

func checkSparkUI(appName string, sparkApp utils.SparkJob, suite *SparkEncryptionSuite) {
	if err := suite.operator.WaitForJobState(sparkApp, v1beta2.RunningState); err != nil {
		suite.Fail("SparkApplication \"%s\" is not running", appName, err)
	}
	if err := utils.RetryWithTimeout(20*time.Second, 2*time.Second, func() error {
		response, err := utils.Kubectl("exec", utils.DriverPodName(appName), "-n", sparkApp.Namespace,
			"--",
			"curl",
			"--insecure", // allow insecure SSL
			"--location", // follow redirects
			"--include",  //include headers
			"https://localhost:4440")
		if err != nil {
			return err
		}
		if !strings.Contains(response, "HTTP/1.1 200") {
			return fmt.Errorf("received status code is not successful")
		}
		suite.Assert().Contains(response, "<title>MockTaskRunner - Spark Jobs</title>")
		return nil
	}); err != nil {
		suite.Fail("Unable to access Spark UI", err)
	}
}
