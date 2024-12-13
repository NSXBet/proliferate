package apply

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var templateFuncs = template.FuncMap{
	"splitLast": func(sep, s string) string {
		parts := strings.Split(s, sep)
		return parts[len(parts)-1]
	},
	"lower": strings.ToLower,
}

func renderTemplate(name string, tmpl string, values interface{}) (string, error) {
	t, err := template.New(name).Funcs(templateFuncs).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Values": values,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return buf.String(), nil
}
