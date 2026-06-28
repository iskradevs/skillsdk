package codegen

import (
	"embed"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var boilerplateTmpl = template.Must(
	template.New("boilerplate.tmpl").ParseFS(templatesFS, "templates/boilerplate.tmpl"),
)

type boilerplateData struct {
	Package  string
	Deferred []string
}
