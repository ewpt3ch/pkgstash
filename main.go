package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const repoRoot = "/home/ewpt3ch/dev/pacman-cache-server/tmprepo"

func main() {
	const port = "8090"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{repo}/os/{arch}/{file}", handlePackage)

	httpServe := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Fatal(httpServe.ListenAndServe())

}

func handlePackage(w http.ResponseWriter, req *http.Request) {
	repo := req.PathValue("repo")
	arch := req.PathValue("arch")
	file := req.PathValue("file")

	filePath := filepath.Join(repoRoot, repo, "os", arch, file)
	// is where we handle cache misses and fetch the file
	// from a mirror, for now we send a 404
	if _, err := os.Stat(filePath); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte("No such file"))
		return
	}

	http.ServeFile(w, req, filePath)
}
