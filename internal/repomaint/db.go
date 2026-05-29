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
)

func (r *RepoSync) buildMap(repo string) (map[string]string, error) {
	// create slice all filenames in repo
	pkgs, err := r.getCachedPkgs(repo)
	if err != nil {
		return nil, err
	}

	// open db file
	db, fr, gzr, err := r.openDb(repo)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer fr.Close()
	//nolint:errcheck
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
		pkgName, pkgFile, err := parseDesc(db)
		if err != nil {
			slog.Warn("failed to parse desc file", "pkg", hdr.Name, "err", err)
			continue
		}
		if !slices.Contains(pkgs, pkgFile) {
			continue
		}

		pkgMap[pkgName] = pkgFile
	}

	return pkgMap, nil
}

func (r *RepoSync) updatablePkgs(repo string, pkgs map[string]string) ([]string, error) {

	// open db file
	db, fr, gzr, err := r.openDb(repo)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer fr.Close()
	//nolint:errcheck
	defer gzr.Close()

	files := []string{}
	// for entry in db
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
		pkgName, currentFileName, err := parseDesc(db)
		if err != nil {
			slog.Warn("failed to parse desc file", "pkg", hdr.Name, "err", err)
			continue
		}
		// check if pkg is cached, continue if not
		cachedFileName, ok := pkgs[pkgName]
		if !ok {
			continue
		}
		// pkg is cached, check if needs update
		if cachedFileName == currentFileName {
			continue
		}
		files = append(files, currentFileName)

	}
	return files, nil
}

func (r *RepoSync) getCachedPkgs(repo string) ([]string, error) {
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
		pkgs = append(pkgs, f.Name())
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
		if err := f.Close(); err != nil {
			slog.Warn("failed to close gzip reader", "err", err)
		}
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
