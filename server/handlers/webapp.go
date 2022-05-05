package handlers

import (
	"context"
	"flag"
	"github.com/julienschmidt/httprouter"
	"io/fs"
	"net/http"
	"os"
)

var webBuild = flag.String("web_build", "", "`build` folder for web app")

func createWebApp(ctx context.Context, r *httprouter.Router) {
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
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, *webBuild+"/index.html")
	})
	if err != nil {
		panic(err)
	}
}
