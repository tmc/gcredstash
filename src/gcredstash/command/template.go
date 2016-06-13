package command

import (
	"bytes"
	"fmt"
	"gcredstash"
	"html/template"
	"io/ioutil"
	"os"
	"strings"
)

type TemplateCommand struct {
	Meta
}

func (c *TemplateCommand) parseArgs(args []string) (string, bool, error) {
	newArgs, inPlace := gcredstash.HasOption(args, "-i")

	if len(newArgs) < 1 {
		return "", false, fmt.Errorf("too few arguments")
	}

	if len(newArgs) > 1 {
		return "", false, fmt.Errorf("too few arguments")
	}

	tmplFile := newArgs[0]

	return tmplFile, inPlace, nil
}

func (c *TemplateCommand) readTemplate(filename string) (string, error) {
	var content string

	if filename == "-" {
		content = gcredstash.ReadStdin()
	} else {
		var err error
		content, err = gcredstash.ReadFile(filename)

		if err != nil {
			return "", nil
		}
	}

	return content, nil
}

func (c *TemplateCommand) getCredential(credential string, context map[string]string) (string, error) {
	value, err := c.Driver.GetSecret(credential, "", c.Table, context)

	if err != nil {
		return "", err
	}

	return value, nil
}

func (c *TemplateCommand) executeTemplate(name string, content string) (string, error) {
	tmpl := template.New(name)

	tmpl = tmpl.Funcs(template.FuncMap{
		"get": func(args ...interface{}) string {
			if len(args) < 1 {
				return "(error: too few arguments)"
			}

			newArgs := []string{}

			for _, arg := range args {
				str, ok := arg.(string)

				if !ok {
					return fmt.Sprintf("(error: cannot cast %v to string)", arg)
				}

				newArgs = append(newArgs, str)
			}

			credential := newArgs[0]
			context, err := gcredstash.ParseContext(newArgs[1:])

			if err != nil {
				return fmt.Sprintf("(error: %s)", err.Error())
			}

			value, err := c.getCredential(credential, context)

			if err != nil {
				return fmt.Sprintf("(error: %s)", err.Error())
			}

			return value
		},
		"env": func(args ...interface{}) string {
			if len(args) < 1 {
				return "(error: too few arguments)"
			}

			if len(args) > 1 {
				return "(error: too many arguments)"
			}

			key, ok := args[0].(string)

			if !ok {
				return fmt.Sprintf("(error: cannot cast %v to string)", args[0])
			}

			return os.Getenv(key)
		},
	})

	tmpl, err := tmpl.Parse(content)

	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	tmpl.Execute(buf, nil)

	return buf.String(), nil
}

func (c *TemplateCommand) RunImpl(args []string) (string, error) {
	tmplFile, inPlace, err := c.parseArgs(args)

	if err != nil {
		return "", err
	}

	tmplContent, err := c.readTemplate(tmplFile)

	if err != nil {
		return "", err
	}

	out, err := c.executeTemplate(tmplFile, tmplContent)

	if inPlace {
		err = ioutil.WriteFile(tmplFile, []byte(out), 0644)
		out = ""
	}

	return out, err
}

func (c *TemplateCommand) Run(args []string) int {
	out, err := c.RunImpl(args)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return 1
	}

	fmt.Print(out)

	return 0
}

func (c *TemplateCommand) Synopsis() string {
	return "Parse a template file with credentials"
}

func (c *TemplateCommand) Help() string {
	helpText := `
usage: gcredstash template [-i] template_file
`
	return strings.TrimSpace(helpText)
}