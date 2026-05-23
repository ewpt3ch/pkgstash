package repomaint

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func (r *RepoSync) buildMap(repo string) (map[string]string, error) {
	// create slice all filenames in repo
	pkgs, err := r.cachedPkgs(repo)
	if err != nil {
		return nil, err
	}

	// open db file
	db, fr, gzr, err := r.openDb(repo)
	if err != nil {
		return nil, err
	}
	defer fr.Close()
	defer gzr.Close()

	// for entry in db
	pkgMap := make(map[string]string)
	for {
		hdr, err := db.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != "desc" {
			continue
		}
		if !slices.Contains(pkgs, filepath.Dir(hdr.Name)) {
			continue
		}
		pkgName, pkgFile, err := parseDesc(db)
		if err != nil {
			slog.Warn("failed to parse desc file", "pkg", hdr.Name, "err", err)
			continue
		}

		pkgMap[pkgName] = pkgFile
	}

	return pkgMap, nil
}

func (r *RepoSync) cachedPkgs(repo string) ([]string, error) {
	repoPath := filepath.Join(repo, repoArch)
	f, err := r.root.Open(repoPath)
	if err != nil {
		return nil, err
	}
	files, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, f := range files {
		pkg := strings.TrimSuffix(f.Name(), pkgSuffix)
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

func (r *RepoSync) openDb(repo string) (*tar.Reader, *os.File, *gzip.Reader, error) {

	repoPath := filepath.Join(repo, repoArch)
	path := filepath.Join(repoPath, repo+dbSuffix)
	f, err := r.root.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, nil, nil, err
	}
	tr := tar.NewReader(gr)

	return tr, f, gr, nil
}

func parseDesc(f io.Reader) (pkgname, filename string, err error) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		switch scanner.Text() {
		case "%NAME%":
			scanner.Scan()
			pkgname = scanner.Text()
		case "%FILENAME%":
			scanner.Scan()
			filename = scanner.Text()
		}
		if pkgname != "" && filename != "" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if pkgname == "" || filename == "" {
		return "", "", fmt.Errorf("incomplete or corrupt desc")
	}

	return pkgname, filename, nil
}
