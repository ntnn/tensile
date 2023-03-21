package tensile

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/ntnn/tensile/facts"
)

type TemplateData struct {
	Facts  facts.Facts
	Custom map[string]any
}

func Template(facts facts.Facts, input string, writer io.Writer, customData map[string]any) error {
	t, err := template.New("").Parse(input)
	if err != nil {
		return fmt.Errorf("tensile: error parsing template: %w", err)
	}

	if err := t.Execute(writer, TemplateData{
		Facts:  facts,
		Custom: customData,
	}); err != nil {
		return fmt.Errorf("tensile: error executing template: %w", err)
	}

	return nil
}

func TemplateString(facts facts.Facts, input string, customData map[string]any) (string, error) {
	b := &bytes.Buffer{}

	if err := Template(facts, input, b, customData); err != nil {
		return "", err
	}

	return b.String(), nil
}
