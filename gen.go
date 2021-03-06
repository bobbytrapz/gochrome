// +build ignore

// This program generates protocol.go
// using the installed version of chrome
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/bobbytrapz/gochrome"
)

func main() {
	fmt.Fprintf(os.Stderr, "[*] Generating protocol.go\n")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.New(os.Stderr, "gochrome: ", log.LstdFlags|log.Lshortfile)
	gochrome.Log = logger.Printf

	browser := gochrome.NewBrowser()

	_, err := browser.Start(ctx, "", 32719)
	if err != nil {
		panic(err)
	}

	defer browser.Wait()

	protocol := browser.GetProtocol()
	cancel()

	data := protocoldata{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   protocol.VersionString(),
	}

	for _, domain := range protocol.Domains {
		// domain types
		for _, t := range domain.Types {
			props := cleanProperties(domain, t.Properties, t.ID)

			if t.Type == "object" && len(props) == 0 {
				t.Type = "map[string]interface{}"
			}

			td := gochrome.Type{
				ID:          fmt.Sprintf("%s%s", domain.Domain, t.ID),
				Type:        typeReplacer.Replace(t.Type),
				Description: t.Description,
				Properties:  props,
			}

			data.Types = append(data.Types, td)
			// fmt.Fprintf(os.Stderr, "type %+v\n", td)
		}

		// domain commands
		for _, c := range domain.Commands {
			var cmd gochrome.Command
			name := fmt.Sprintf("%s%s", domain.Domain, strings.Title(c.Name))
			cmd.Method = fmt.Sprintf("%s.%s", domain.Domain, c.Name)
			params := cleanParameters(domain, c.Parameters, strings.Title(c.Name))
			returns := cleanReturns(domain, c.Returns, strings.Title(c.Name))

			cmd.Name = name
			cmd.Description = c.Description
			cmd.Parameters = params
			cmd.Returns = returns
			data.Commands = append(data.Commands, cmd)
			// fmt.Fprintf(os.Stderr, "cmd %+v\n", cmd)
		}

		// domain events
		for _, e := range domain.Events {
			data.Events = append(data.Events, gochrome.Event{
				Name:        fmt.Sprintf("%s%s", strings.Title(domain.Domain), strings.Title(e.Name)),
				Description: e.Description,
				Parameters:  cleanParameters(domain, e.Parameters, strings.Title(e.Name)),
				EventName:   fmt.Sprintf("%s.%s", strings.Title(domain.Domain), e.Name),
			})

			// fmt.Fprintf(os.Stderr, "%+v\n", data.Events[len(data.Events)-1])
		}
	}

	var buf bytes.Buffer
	protocolTmpl.Execute(&buf, data)
	if true {
		ioutil.WriteFile("protocol.go", buf.Bytes(), 0664)
	} else {
		fmt.Fprintf(os.Stderr, "%s", buf.Bytes())
	}
}

var typeReplacer = strings.NewReplacer(
	"integer", "int",
	"object", "struct",
	"array", "[]interface{}",
	"number", "float64",
	"any", "interface{}",
	"boolean", "bool",
	"binary", "string",
	".", "",
)

var nameReplacer = strings.NewReplacer(
	"range", "Range",
	"type", "Type",
)

var funcMap = template.FuncMap{
	"Title": strings.Title,
}

var protocolTmpl = template.Must(template.New("").Funcs(funcMap).Parse(`// Code generated by go generate; DO NOT EDIT.
// Chrome protocol v{{ .Version }}
// {{ .Timestamp }}
package gochrome

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func isZero(v interface{}) bool {
	vType := reflect.TypeOf(v)
	vZero := reflect.Zero(vType)
	return v == nil || reflect.DeepEqual(v, vZero.Interface())
}

{{ range .Types }}
type {{.ID}} {{.Type}}{{ if .Properties }} { {{ range .Properties }}
	{{if .Description}}/* {{ .Description }} */
	{{end}}{{ .Name | Title }} {{ .Type }}{{ end }}
}{{ end }}
{{ end }}
{{ range .Commands }}
type {{.Name}}Returns struct {
	{{ range .Returns }}
	{{.Name | Title}} {{.Type}}
	{{ end }}
}

/* {{.Description}} */
func (t *Tab) {{.Name}}({{range $ndx, $p := .Parameters}}{{if $ndx}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) ({{.Name}}Returns, error) {
	params_ := make(map[string]interface{})

	{{ range .Parameters }}
	{{ if .Optional }}
	if !isZero({{.Name}}) {
		params_["{{.Name}}"] = {{.Name}}
	}
	{{ else }}
	params_["{{.Name}}"] = {{.Name}}
	{{ end }}
	{{ end }}

	ch := t.SendCommand(map[string]interface{}{
		"method": "{{.Method}}",
		"params": params_,
	})

	var returns_ {{.Name}}Returns
	data_ := <-ch
	err_ := json.Unmarshal(data_, &returns_)
	if err_ != nil {
		return returns_, fmt.Errorf("json.Unmarshal: %w", err_)
	}

	return returns_, nil
}
{{ end }}
/* Event Handlers */
{{ range .Events }}
type {{.Name}}Event struct {
	{{ range .Parameters }}
	{{.Name | Title}} {{.Type}}
	{{ end }}
}
type {{.Name | Title}}Handler func (ev {{.Name | Title}}Event)
{{ end }}
/* Handle Tab Events */
type tabEventHandlers struct {
{{ range .Events }}
	On{{.Name | Title}} {{.Name | Title}}Handler
{{ end }}
}
func (t *Tab) HandleEvent(method string, params json.RawMessage) error {
	switch method {
{{ range .Events }}
	case "{{.EventName}}":
		var ev {{.Name | Title}}Event
		err := json.Unmarshal(params, &ev)
		if err != nil {
			Log("{{.EventName}}: %s", err)
			return err
		}
		if t.Events.On{{.Name | Title}} != nil {
			go t.Events.On{{.Name | Title}}(ev)
		}
{{ end }}
	default:
		return errEventNotHandled
	}
	return nil
}

`))

type protocoldata struct {
	Timestamp string
	Version   string
	Types     []gochrome.Type
	Commands  []gochrome.Command
	Events    []gochrome.Event
}

// helpers

// clean* helpers
// prefix domain name to name
// resolve $ref to actual types
// change JavaScript types to Go types

func cleanProperties(domain gochrome.Domain, props []gochrome.Property, name string) (cleaned []gochrome.Property) {
	// prefix domain name
	name = fmt.Sprintf("%s%s", domain.Domain, name)
	for _, p := range props {
		var pt string

		if p.Ref != "" {
			pt = p.Ref
			if !strings.Contains(pt, ".") {
				pt = fmt.Sprintf("%s%s", domain.Domain, pt)
			}
		} else {
			pt = p.Type
		}

		if pt == "array" {
			var at string
			if p.Items.Ref != "" {
				at = p.Items.Ref
				if !strings.Contains(at, ".") {
					at = fmt.Sprintf("%s%s", domain.Domain, at)
				}
			} else {
				at = p.Items.Type
				if at == "object" {
					at = "map[string]interface{}"
				}
			}
			pt = fmt.Sprintf("[]%s", at)
		}

		if pt == "object" {
			pt = "map[string]interface{}"
		}

		if pt == name {
			pt = fmt.Sprintf("*%s", pt)
		}

		props = append(props, gochrome.Property{
			Name:        nameReplacer.Replace(p.Name),
			Description: p.Description,
			Optional:    p.Optional,
			Type:        typeReplacer.Replace(pt),
		})
		// fmt.Fprintf(os.Stderr, "prop %+v\n", props[len(props)-1])
	}

	return
}

func cleanParameters(domain gochrome.Domain, params []gochrome.Parameter, name string) (cleaned []gochrome.Parameter) {
	// prefix domain name
	name = fmt.Sprintf("%s%s", domain.Domain, name)
	for _, p := range params {
		var pt string

		if p.Ref != "" {
			pt = p.Ref
			if !strings.Contains(pt, ".") {
				pt = fmt.Sprintf("%s%s", domain.Domain, pt)
			}
		} else {
			pt = p.Type
		}

		if pt == "array" {
			var at string
			if p.Items.Ref != "" {
				at = p.Items.Ref
				if !strings.Contains(at, ".") {
					at = fmt.Sprintf("%s%s", domain.Domain, at)
				}
			} else {
				at = p.Items.Type
				if at == "object" {
					at = "map[string]interface{}"
				}
			}
			pt = fmt.Sprintf("[]%s", at)
		}

		if pt == "object" {
			pt = "map[string]interface{}"
		}

		if pt == name {
			pt = fmt.Sprintf("*%s", pt)
		}

		cleaned = append(cleaned, gochrome.Parameter{
			Name:     nameReplacer.Replace(p.Name),
			Type:     typeReplacer.Replace(pt),
			Optional: p.Optional,
		})
	}

	return
}

func cleanReturns(domain gochrome.Domain, returns []gochrome.Return, name string) (cleaned []gochrome.Return) {
	// prefix domain name
	name = fmt.Sprintf("%s%s", domain.Domain, name)
	for _, p := range returns {
		var pt string

		if p.Ref != "" {
			pt = p.Ref
			if !strings.Contains(pt, ".") {
				pt = fmt.Sprintf("%s%s", domain.Domain, pt)
			}
		} else {
			pt = p.Type
		}

		if pt == "array" {
			var at string
			if p.Items.Ref != "" {
				at = p.Items.Ref
				if !strings.Contains(at, ".") {
					at = fmt.Sprintf("%s%s", domain.Domain, at)
				}
			} else {
				at = p.Items.Type
			}
			pt = fmt.Sprintf("[]%s", at)
		}

		if pt == "object" {
			pt = "map[string]interface{}"
		}

		if pt == "[]object" {
			pt = "[]map[string]interface{}"
		}

		if pt == name {
			pt = fmt.Sprintf("*%s", pt)
		}

		cleaned = append(cleaned, gochrome.Return{
			Name:     nameReplacer.Replace(p.Name),
			Type:     typeReplacer.Replace(pt),
			Optional: p.Optional,
		})
	}

	return
}
