package utils

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"text/template"
)

const sparkRbacTemplateName string = "spark-rbac.yaml.template"

var templates *template.Template

func init() {
	var err error
	templates, err = template.ParseGlob(path.Join(TestDir, "templates/*.template"))

	if err != nil {
		log.Fatal("Can't parse templates")
		panic(err)
	}
}

/* Creates a spark operator RBAC file from a template for specified namespace name. Returns a path to the file.
   Do not forget to remove it later with os.Remove() !!! */
func createSparkOperatorNamespace(namespace string) string {
	file, err := ioutil.TempFile("/tmp", "spark-test-")
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	err = templates.ExecuteTemplate(file, sparkRbacTemplateName, map[string]string{"Namespace": namespace})
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	return file.Name()
}
