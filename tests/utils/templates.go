package utils

import (
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
