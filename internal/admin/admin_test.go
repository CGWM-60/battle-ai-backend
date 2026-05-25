package admin

import (
	"html/template"
	"testing"
)

func TestAdminTemplatesParse(t *testing.T) {
	if _, err := template.New("admin").Funcs(adminTemplateFuncs()).Parse(adminHTML); err != nil {
		t.Fatalf("admin templates should parse: %v", err)
	}
}
