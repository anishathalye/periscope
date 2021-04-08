// Copyright (c) 2020-2021 Anish Athalye. Released under GPLv3.

package periscope

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/herror"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/afero"
	"golang.org/x/crypto/ssh/terminal"
)

const dbDirectory = "periscope"
const dbName = "periscope.sqlite"

type Periscope struct {
	fs        afero.Fs
	realFs    bool
	db        *db.Session
	dbPath    string
	outStream io.Writer
	errStream io.Writer
	options   *Options
}

type Options struct {
	Debug bool
}

func dbPath() (string, herror.Interface) {
	cacheDirRoot, err := os.UserCacheDir()
	if err != nil {
		return "", herror.Unlikely(err, "unable to determine cache directory", `
Ensure that $HOME or $XDG_CACHE_HOME is set.
		`)
	}
	cacheDir := filepath.Join(cacheDirRoot, dbDirectory)
	err = os.MkdirAll(cacheDir, 0o755)
	if err != nil {
		return "", herror.Unlikely(err, fmt.Sprintf("unable to create cache directory '%s'", cacheDir), fmt.Sprintf(`
Ensure that the user cache directory '%s' exists and is writable.
		`, cacheDirRoot))
	}
	dbPath := filepath.Join(cacheDir, dbName)
	return dbPath, nil
}

func New(options *Options) (*Periscope, herror.Interface) {
	dbPath, err := dbPath()
	if err != nil {
		return nil, err
	}
	db, err := db.New(dbPath)
	if err != nil {
		return nil, err
	}
	fs := afero.NewOsFs()
	return &Periscope{
		fs:        fs,
		realFs:    true,
		db:        db,
		dbPath:    dbPath,
		outStream: os.Stdout,
		errStream: os.Stderr,
		options:   options,
	}, nil
}

func (ps *Periscope) progressBar(total int, template string) *pb.ProgressBar {
	bar := pb.New(total)
	bar.SetRefreshRate(25 * time.Millisecond)
	bar.SetTemplateString(template)
	bar.SetMaxWidth(99)
	bar.Start()
	if w, ok := ps.errStream.(*os.File); ok && terminal.IsTerminal(int(w.Fd())) {
		bar.SetWriter(ps.errStream)
	} else {
		bar.SetWriter(ioutil.Discard)
	}
	return bar
}
