// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"text/template"

	lic "github.com/elastic/elastic-agent/dev-tools/licenses"
	"github.com/elastic/elastic-agent/pkg/packer"
)

var (
	input   string
	output  string
	license string
)

func init() {
	flag.StringVar(&input, "in", "", "Source of input. \"-\" means reading from stdin")
	if flag.Lookup("out") == nil {
		flag.StringVar(&output, "out", "-", "Output path. \"-\" means writing to stdout")
	}
	flag.StringVar(&license, "license", "Elastic", "License header for generated file.")
}

var tmplPgp = template.Must(template.New("pgp").Parse(`
{{ .License }}
// Code generated by /dev-tools/cmd/buildspec/buildPgp.go - DO NOT EDIT.

package release

import (
	"github.com/elastic/elastic-agent/pkg/packer"
)

// pgp bytes is a packed in public gpg key
var pgpBytes []byte

func init() {
	// Packed Files
	{{ range $i, $f := .Files -}}
	// {{ $f }}
	{{ end -}}
	pgpBytes = packer.MustUnpack("{{ .Pack }}")["GPG-KEY-elasticsearch"]
}

// PGP return pgpbytes and a flag describing whether or not no pgp is valid.
func PGP() (bool, []byte) {
	return allowEmptyPgp == "true", pgpBytes
}
`))

func main() {
	flag.Parse()

	if len(input) == 0 {
		fmt.Fprintln(os.Stderr, "Invalid input source")
		os.Exit(1)
	}

	l, err := lic.Find(license)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem to retrieve the license, error: %+v", err)
		os.Exit(1)
		return
	}

	data, err := gen(input, l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while generating the file, err: %+v\n", err)
		os.Exit(1)
	}

	if output == "-" {
		os.Stdout.Write(data)
		return
	} else {
		ioutil.WriteFile(output, data, 0640)
	}

	return
}

func gen(path string, l string) ([]byte, error) {
	pack, files, err := packer.Pack(input)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	tmplPgp.Execute(&buf, struct {
		Pack    string
		Files   []string
		License string
	}{
		Pack:    pack,
		Files:   files,
		License: l,
	})

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return formatted, nil
}