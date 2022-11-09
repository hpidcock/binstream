package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type Stream struct {
	ContentID string             `json:"content_id"`
	Updated   string             `json:"updated"`
	Format    string             `json:"format"`
	DataType  string             `json:"datatype"`
	Products  map[string]Product `json:"products"`
}

type Product struct {
	DataType string             `json:"datatype"`
	Format   string             `json:"format"`
	Arch     string             `json:"arch"`
	FileType string             `json:"ftype"`
	Release  string             `json:"release"`
	Versions map[string]Version `json:"versions"`
}

type Version struct {
	Updated string            `json:"updated"`
	Items   map[string]Binary `json:"items"`
}

type Binary struct {
	Path     string `json:"path"`
	Size     int    `json:"size"`
	SHA256   string `json:"sha256"`
	MD5      string `json:"md5"`
	Arch     string `json:"arch"`
	FileType string `json:"ftype"`
	Release  string `json:"release"`
	Version  string `json:"version"`
}

func generateStream(s stream) (*Stream, error) {
	res := &Stream{
		ContentID: s.fullName,
		DataType:  "content-download",
		Format:    "products:1.0",
		Updated:   time.Now().Format(time.RFC1123Z),
		Products:  map[string]Product{},
	}
	date := time.Now().Format("20060102")

	bins, err := ioutil.ReadDir(filepath.Join(*flagDir, s.name))
	if os.IsNotExist(err) {
		return res, nil
	} else if err != nil {
		return nil, err
	}
	for _, f := range bins {
		if f.IsDir() {
			continue
		}
		version, osName, archName, ok := matchBinName(f.Name())
		if !ok {
			continue
		}

		file, err := os.Open(filepath.Join(*flagDir, s.name, f.Name()))
		if err != nil {
			return nil, err
		}

		sha256sum := sha256.New()
		md5sum := md5.New()
		w := io.MultiWriter(sha256sum, md5sum)
		_, err = io.Copy(w, file)
		if err != nil {
			return nil, err
		}
		file.Close()
		sha256string := hex.EncodeToString(sha256sum.Sum(nil))
		md5string := hex.EncodeToString(md5sum.Sum(nil))

		for _, p := range platforms {
			if p.os != osName {
				continue
			}
			productName := executeTemplate(s.productNameTemplate, map[string]string{
				"series": p.series,
				"arch":   ubuntuArch(archName),
			})
			product, ok := res.Products[productName]
			if !ok {
				product = Product{
					DataType: "content-download",
					Format:   "products:1.0",
					FileType: "tar.gz",
					Release:  p.release,
					Arch:     ubuntuArch(archName),
					Versions: map[string]Version{
						date: Version{
							Updated: time.Now().Format(time.RFC1123Z),
							Items:   map[string]Binary{},
						},
					},
				}
			}
			binName := executeTemplate(s.versionNameTemplate, map[string]string{
				"version": version,
				"release": p.release,
				"arch":    ubuntuArch(archName),
			})
			product.Versions[date].Items[binName] = Binary{
				Path:     fmt.Sprintf("%s/%s", s.key, f.Name()),
				Size:     int(f.Size()),
				SHA256:   sha256string,
				MD5:      md5string,
				FileType: "tar.gz",
				Release:  p.release,
				Arch:     ubuntuArch(archName),
				Version:  version,
			}
			res.Products[productName] = product
		}
	}

	return res, nil
}
