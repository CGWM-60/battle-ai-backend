package admin

import (
	"html/template"
	"testing"
)

func TestAdminTemplatesParse(t *testing.T) {
	if _, err := template.New("admin").Funcs(template.FuncMap{"json": toJSON}).Parse(adminHTML); err != nil {
		t.Fatalf("admin templates should parse: %v", err)
	}
}
