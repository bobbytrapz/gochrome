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

	fmt.Fprintf(os.Stderr, "Chrome protocol version: %s\n", protocol.VersionString())

	data := protocoldata{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	for _, domain := range protocol.Domains {
		// protocol types
		for _, t := range domain.Types {
			name := fmt.Sprintf("%s%s", domain.Domain, t.ID)
			var props []gochrome.Property
			for _, p := range t.Properties {
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

				if pt == name {
					pt = fmt.Sprintf("*%s", pt)
				}

				props = append(props, gochrome.Property{
					Name:        nameReplacer.Replace(p.Name),
					Description: p.Description,
					Optional:    p.Optional,
					Type:        typeReplacer.Replace(pt),
				})
			}

			if t.Type == "object" && len(props) == 0 {
				t.Type = "map[string]interface{}"
			}

			td := gochrome.Type{
				ID:          name,
				Type:        typeReplacer.Replace(t.Type),
				Description: t.Description,
				Properties:  props,
			}

			data.Types = append(data.Types, td)
		}

		// protocol commands
		for _, c := range domain.Commands {
			var cmd gochrome.Command
			name := fmt.Sprintf("%s%s", domain.Domain, strings.Title(c.Name))
			cmd.Method = fmt.Sprintf("%s.%s", domain.Domain, c.Name)
			var params []gochrome.Parameter
			for _, p := range c.Parameters {
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

				if pt == name {
					pt = fmt.Sprintf("*%s", pt)
				}

				params = append(params, gochrome.Parameter{
					Name:     nameReplacer.Replace(p.Name),
					Type:     typeReplacer.Replace(pt),
					Optional: p.Optional,
				})
			}

			var returns []gochrome.Return
			for _, p := range c.Returns {
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

				returns = append(returns, gochrome.Return{
					Name:     nameReplacer.Replace(p.Name),
					Type:     typeReplacer.Replace(pt),
					Optional: p.Optional,
				})
			}

			cmd.Name = name
			cmd.Description = c.Description
			cmd.Parameters = params
			cmd.Returns = returns
			data.Commands = append(data.Commands, cmd)
		}
	}

	var buf bytes.Buffer
	protocolTmpl.Execute(&buf, data)
	ioutil.WriteFile("protocol.go", buf.Bytes(), 0664)
}

var typeReplacer = strings.NewReplacer(
	"integer", "int",
	"object", "struct",
	"array", "[]interface{}",
	"number", "float64",
	"any", "interface{}",
	"boolean", "bool",
	"binary", "[]byte",
	".", "",
)

var nameReplacer = strings.NewReplacer(
	"range", "_range",
	"type", "_type",
)

var funcMap = template.FuncMap{
	"Title": strings.Title,
}

var protocolTmpl = template.Must(template.New("").Funcs(funcMap).Parse(`// Code generated by go generate; DO NOT EDIT.
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

	t.SendCommand(map[string]interface{}{
		"method": "{{.Method}}",
		"params": params_,
	})

	var returns_ {{.Name}}Returns
	data_ := <-t.recv
	err_ := json.Unmarshal(data_, &returns_)
	if err_ != nil {
		return returns_, fmt.Errorf("json.Unmarshal: %w", err_)
	}

	return returns_, nil
}
{{ end }}
`))

type protocoldata struct {
	Timestamp string
	Types     []gochrome.Type
	Commands  []gochrome.Command
}
