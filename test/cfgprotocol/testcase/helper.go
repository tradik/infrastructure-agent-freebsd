package testcase

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/sirupsen/logrus"
)

func createFile(outputDir string, filename string, templatePath string, vars map[string]interface{}) error {
	t := template.Must(template.ParseFiles(templatePath))
	outputPath := filepath.Join(outputDir, filename)
	f, err := os.Create(outputPath)
	if err != nil {
		logrus.Println("create file: ", err)
		return err
	}
	if err := t.Execute(f, vars); err != nil {
		return err
	}
	f.Close()
	return nil
}
