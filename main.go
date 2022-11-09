package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/fujiwara/shapeio"
	"github.com/gorilla/mux"
)

var (
	flagListen    = flag.String("listen", ":9999", "address to listen simplestreams server on")
	flagDir       = flag.String("dir", mustGetwd(), "directory to find streams binaries")
	flagRateLimit = flag.Int("rate", 0, "kb/s rate limit - default to unlimited")
)

type platform struct {
	os      string
	series  string
	release string
}

var platforms = []platform{
	// {"linux", "20.04", "focal"},
	// {"linux", "19.10", "eoan"},
	// {"linux", "19.04", "disco"},
	// {"linux", "18.04", "bionic"},
	// {"linux", "16.04", "xenial"},
	{"linux", "ubuntu", "ubuntu"},
	{"linux", "centos", "centos"},
}

type stream struct {
	key                 string
	name                string
	fullName            string
	urlName             string
	productNameTemplate *template.Template
	versionNameTemplate *template.Template
}

var streams = map[string]stream{
	"released-tools": {
		key:                 "released-tools",
		name:                "released",
		fullName:            "com.ubuntu.juju:released:tools",
		urlName:             "com.ubuntu.juju-released-tools",
		productNameTemplate: template.Must(template.New("").Parse("com.ubuntu.juju:{{.series}}:{{.arch}}")),
		versionNameTemplate: template.Must(template.New("").Parse("{{.version}}-{{.release}}-{{.arch}}")),
	},
	"proposed-tools": {
		key:                 "proposed-tools",
		name:                "proposed",
		fullName:            "com.ubuntu.juju:proposed:tools",
		urlName:             "com.ubuntu.juju-proposed-tools",
		productNameTemplate: template.Must(template.New("").Parse("com.ubuntu.juju:{{.series}}:{{.arch}}")),
		versionNameTemplate: template.Must(template.New("").Parse("{{.version}}-{{.release}}-{{.arch}}")),
	},
	"proposed-agents": {
		key:                 "proposed-agents",
		name:                "proposed",
		fullName:            "com.ubuntu.juju:proposed:agents",
		urlName:             "com.ubuntu.juju-proposed-agents",
		productNameTemplate: template.Must(template.New("").Parse("com.ubuntu.juju:{{.series}}:{{.arch}}")),
		versionNameTemplate: template.Must(template.New("").Parse("{{.version}}-{{.release}}-{{.arch}}")),
	},
	"released-agents": {
		key:                 "released-agents",
		name:                "released",
		fullName:            "com.ubuntu.juju:released:agents",
		urlName:             "com.ubuntu.juju-released-agents",
		productNameTemplate: template.Must(template.New("").Parse("com.ubuntu.juju:{{.series}}:{{.arch}}")),
		versionNameTemplate: template.Must(template.New("").Parse("{{.version}}-{{.release}}-{{.arch}}")),
	},
}

func main() {
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/streams/v1/index.json", IndexHandler).Methods("GET")
	r.HandleFunc("/streams/v1/index2.json", IndexAllHandler).Methods("GET")
	for _, v := range streams {
		r.HandleFunc(fmt.Sprintf("/streams/v1/%s.json", v.urlName), StreamHandler(v)).Methods("GET")
	}
	r.HandleFunc("/{stream}/{file}", FileHandler).Methods("GET")

	err := http.ListenAndServe(*flagListen, r)
	if err != nil {
		panic(err)
	}
}

func mustGetwd() string {
	s, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return s
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s %s\n", r.Method, r.RequestURI)
	idx, err := generateIndex([]stream{streams["released-tools"]})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(idx)
}

func IndexAllHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s %s\n", r.Method, r.RequestURI)
	s := []stream{}
	for _, v := range streams {
		s = append(s, v)
	}
	idx, err := generateIndex(s)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(idx)
}

func StreamHandler(st stream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.RequestURI)
		res, err := generateStream(st)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}

func FileHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s %s\n", r.Method, r.RequestURI)
	vars := mux.Vars(r)
	s, _ := vars["stream"]
	stream, ok := streams[s]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	f, _ := vars["file"]
	f = filepath.Clean(f)
	f = filepath.Join(*flagDir, stream.name, f)

	file, err := os.Open(f)
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	w.Header().Add("Content-Length", strconv.Itoa(int(stat.Size())))
	w.Header().Add("Content-Type", "application/tar+gzip")
	w.WriteHeader(http.StatusOK)

	var reader io.Reader = file
	if flagRateLimit != nil && *flagRateLimit > 0 {
		shapedReader := shapeio.NewReaderWithContext(reader, r.Context())
		shapedReader.SetRateLimit(float64(*flagRateLimit * 1024))
		reader = shapedReader
	}

	io.Copy(w, reader)
}
