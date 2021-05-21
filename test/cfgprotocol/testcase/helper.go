package testcase

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func createFile(outputDir string, filename string, templatePath string, vars map[string]interface{}) error {
	t := template.Must(template.ParseFiles(templatePath))
	outputPath := filepath.Join(outputDir, filename)
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	if err := t.Execute(f, vars); err != nil {
		return err
	}
	f.Close()
	return nil
}

// formatRequest generates ascii representation of a request
func FormatHTTPRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}


	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}