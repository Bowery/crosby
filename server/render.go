package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var TEMPLATE_DIR string = "static"

func execute(name string, data interface{}) (*bytes.Buffer, error) {
	fmt.Println("called execute", name, data)

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	tmplName := name + "-partial"

	t := template.New(tmplName)
	path := dir + "/" + TEMPLATE_DIR + "/" + name + ".html"
	log.Println(path)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	t = t.Funcs(template.FuncMap{
		"yield": func() (string, error) {
			return "", fmt.Errorf("yield called with no layout defined")
		},
	})

	t, err = t.Parse(string(buf))
	if err != nil {
		return nil, err
	}

	outBuf := new(bytes.Buffer)
	return outBuf, t.Execute(outBuf, data)
}

func RenderTemplate(wr io.Writer, name string, data interface{}) error {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf, err := execute(name, data)

			// return safe html here since we are rendering our own template
			fmt.Println("Yield was called")
			return template.HTML(buf.String()), err
		},
	})

	layoutPath := TEMPLATE_DIR + "/layout.html"
	buf, err := ioutil.ReadFile(layoutPath)
	if err != nil {
		return err
	}

	tmpl, err = tmpl.Parse(string(buf))
	if err != nil {
		return err
	}
	return tmpl.Execute(wr, data)
}
