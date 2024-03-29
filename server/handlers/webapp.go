package handlers

import (
	"context"
	"flag"
	"io/fs"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

var webBuild = flag.String("web_build", "", "`build` folder for web app")

func serveIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if *webBuild == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, *webBuild+"/index.html")
}

func createWebApp(_ context.Context, r Router) {
	if *webBuild == "" {
		return
	}
	err := fs.WalkDir(os.DirFS(*webBuild), ".",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			urlPath := "/" + path
			filePath := *webBuild + "/" + path

			r.GET(urlPath, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
				http.ServeFile(w, r, filePath)
			})
			return nil
		},
	)
	r.GET("/", serveIndex)
	if err != nil {
		panic(err)
	}
}
