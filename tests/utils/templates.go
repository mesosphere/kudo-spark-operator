package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"text/template"
)

var templates *template.Template

func init() {
	var err error
	templates, err = template.ParseGlob(path.Join(TestDir, "templates/*.yaml"))

	if err != nil {
		log.Fatal("Can't parse templates")
		panic(err)
	}

	templatesLogDetails := "Parsed templates:"
	for _, t := range templates.Templates() {
		templatesLogDetails += fmt.Sprintf("\n- %s", t.Name())
	}
	log.Debug(templatesLogDetails)
}

func createSparkJob(job SparkJob) string {
	file, err := ioutil.TempFile("/tmp", "job-")
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	err = templates.ExecuteTemplate(file, job.Template, job)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	return file.Name()
}

func populateYamlTemplate(name string, params map[string]interface{}) (string, error) {
	file, err := ioutil.TempFile("/tmp", "k8s-")
	if err != nil {
		log.Fatalf("Can't create a temporary file for template %s: %s", name, err)
		return "", err
	}

	err = templates.ExecuteTemplate(file, name, params)
	if err != nil {
		log.Fatalf("Can't populate a yaml template %s: %s", name, err)
		return "", err
	}

	return file.Name(), nil
}
