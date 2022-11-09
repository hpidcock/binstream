package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

type Index struct {
	Index   map[string]IndexStream `json:"index"`
	Updated string                 `json:"updated"`
	Format  string                 `json:"format"`
}

type IndexStream struct {
	Updated  string   `json:"updated"`
	Format   string   `json:"format"`
	DataType string   `json:"datatype"`
	Path     string   `json:"path"`
	Products []string `json:"products"`
}

func generateIndex(s []stream) (*Index, error) {
	idx := Index{
		Updated: time.Now().Format(time.RFC1123Z),
		Format:  "index:1.0",
		Index:   map[string]IndexStream{},
	}

	for _, v := range s {
		idxStream := IndexStream{
			Format:   "products:1.0",
			Updated:  time.Now().Format(time.RFC1123Z),
			DataType: "content-download",
			Path:     fmt.Sprintf("streams/v1/%s.json", v.urlName),
		}

		availableProduct := map[string]struct{}{}
		bins, err := ioutil.ReadDir(filepath.Join(*flagDir, v.name))
		if os.IsNotExist(err) {
			idx.Index[v.fullName] = idxStream
			continue
		} else if err != nil {
			return nil, err
		}
		for _, f := range bins {
			if f.IsDir() {
				continue
			}
			_, osName, archName, ok := matchBinName(f.Name())
			if !ok {
				continue
			}
			for _, p := range platforms {
				if p.os != osName {
					continue
				}
				n := executeTemplate(v.productNameTemplate, map[string]string{
					"series": p.series,
					"arch":   ubuntuArch(archName),
				})
				availableProduct[n] = struct{}{}
			}
		}

		for p := range availableProduct {
			idxStream.Products = append(idxStream.Products, p)
		}

		idx.Index[v.fullName] = idxStream
	}

	return &idx, nil
}

func executeTemplate(tmpl *template.Template, v interface{}) string {
	b := &bytes.Buffer{}
	err := tmpl.Execute(b, v)
	if err != nil {
		panic(err)
	}
	return b.String()
}
