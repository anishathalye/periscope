package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/anishathalye/periscope/herror"

	_ "github.com/mattn/go-sqlite3"
)

type DuplicateSet struct {
	Paths []string
	Size  int64
	Tag   int64
}

type DuplicateInfo struct {
	Path  string
	Tag   int64
	Count int64
}

type DuplicateSummary struct {
	Files     int64
	Unique    int64
	Duplicate int64
	Overhead  int64
}

type Session struct {
	db *sql.DB
}

func New(dataSourceName string) (*Session, herror.Interface) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	// execute dummy statement to catch problems with db access
	_, err = db.Exec("")
	if err != nil {
		return nil, herror.Unlikely(err, fmt.Sprintf("unable to access database at '%s'", dataSourceName), `
Ensure that the directory is writable, and if the database file already exists, ensure it is readable and writable.
		`)
	}
	s := &Session{db}
	herr := s.checkVersion()
	if herr != nil {
		return nil, herr
	}
	err = s.initSchema()
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	return s, nil
}

const versionKey = "version"
const version = 1

func (s *Session) checkVersion() herror.Interface {
	// ensure metadata table exists
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS meta
	(
		key     TEXT UNIQUE,
		value   TEXT
	)
	`)
	if err != nil {
		return herror.Internal(err, "")
	}
	row := s.db.QueryRow("SELECT value FROM meta WHERE key = ?", versionKey)
	var dbVersion string
	err = row.Scan(&dbVersion)
	if err == sql.ErrNoRows {
		// okay, we will initialize version
		_, err = s.db.Exec("INSERT INTO meta (key, value) VALUES (?, ?)", versionKey, strconv.Itoa(version))
		if err != nil {
			return herror.Internal(err, "")
		}
		return nil
	}
	// DB has a version, make sure it's the current version
	dbVersionInt, err := strconv.ParseInt(dbVersion, 10, 0)
	if err != nil || dbVersionInt != version {
		return herror.Unlikely(nil, fmt.Sprintf("database version mismatch: expected %d, got %s", version, dbVersion), `
This database was likely produced by an incompatible version of Periscope. Either use a compatible version of Periscope, or delete the database (by running 'psc finish') and try again.
		`)
	}
	// correct version
	return nil
}

func (s *Session) initSchema() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS duplicates
	(
		id      INTEGER PRIMARY KEY,
		path    TEXT UNIQUE,
		size    INTEGER,
		tag     INTEGER
	)
	`)
	return err
}

func (s *Session) Initialize() herror.Interface {
	_, err := s.db.Exec("DROP TABLE IF EXISTS duplicates")
	if err != nil {
		return herror.Internal(err, "")
	}
	_, err = s.db.Exec("VACUUM")
	if err != nil {
		return herror.Internal(err, "")
	}
	err = s.initSchema()
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

func (s *Session) Add(dupe DuplicateSet) herror.Interface {
	return s.AddAll([]DuplicateSet{dupe})
}

func (s *Session) AddAll(dupes []DuplicateSet) herror.Interface {
	c := make(chan DuplicateSet)
	go func() {
		for _, dupe := range dupes {
			c <- dupe
		}
		close(c)
	}()
	return s.AddAllC(c)
}

func (s *Session) AddAllC(dupes <-chan DuplicateSet) herror.Interface {
	tx, err := s.db.Begin()
	if err != nil {
		return herror.Internal(err, "")
	}
	stmt, err := tx.Prepare("INSERT INTO duplicates (path, size, tag) VALUES (?, ?, ?)")
	if err != nil {
		return herror.Internal(err, "")
	}
	defer stmt.Close()
	for set := range dupes {
		for _, dupe := range set.Paths {
			if _, err := stmt.Exec(dupe, set.Size, set.Tag); err != nil {
				return herror.Internal(err, "")
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

func (s *Session) CreateIndexes() herror.Interface {
	// ensuring that an index on tag exists makes a huge difference in
	// performance for commands like ls
	_, err := s.db.Exec("CREATE INDEX IF NOT EXISTS idx_tag ON duplicates (tag)")
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

func (s *Session) AllDuplicatesC() (<-chan DuplicateSet, herror.Interface) {
	results := make(chan DuplicateSet)
	rows, err := s.db.Query("SELECT path, size, tag FROM duplicates ORDER BY size DESC, tag, path")
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	var prevTag int64
	var set DuplicateSet
	go func() {
		for rows.Next() {
			var path string
			var size int64
			var tag int64
			if err := rows.Scan(&path, &size, &tag); err != nil {
				// how should we handle this error that happens in its own goroutine?
				// give up on this row?
				log.Printf("failure while scanning row: %s", err)
				continue
			}
			if tag != prevTag {
				if len(set.Paths) > 0 {
					results <- set
				}
				set = DuplicateSet{}
				set.Size = size
				set.Tag = tag
			}
			prevTag = tag
			set.Paths = append(set.Paths, path)
		}
		rows.Close()
		if len(set.Paths) > 0 {
			results <- set
		}
		close(results)
	}()
	return results, nil
}

func (s *Session) AllDuplicates() ([]DuplicateSet, herror.Interface) {
	var r []DuplicateSet
	c, err := s.AllDuplicatesC()
	if err != nil {
		return nil, err
	}
	for d := range c {
		r = append(r, d)
	}
	return r, nil
}

func (s *Session) Summary() (DuplicateSummary, herror.Interface) {
	row := s.db.QueryRow(`
	WITH sets AS
	(
		SELECT COUNT(*) AS cnt, size FROM duplicates GROUP BY tag
	)
	SELECT COUNT(*), SUM(cnt), SUM((cnt-1)*size) from sets
	`)
	var unique int64
	var files, overhead sql.NullInt64
	if err := row.Scan(&unique, &files, &overhead); err != nil {
		return DuplicateSummary{}, herror.Internal(err, "")
	}
	return DuplicateSummary{
		Files:     files.Int64,
		Unique:    unique,
		Duplicate: files.Int64 - unique,
		Overhead:  overhead.Int64,
	}, nil
}

func (s *Session) Lookup(path string) (DuplicateSet, herror.Interface) {
	var set DuplicateSet
	row := s.db.QueryRow("SELECT tag FROM duplicates WHERE path = ?", path)
	var tag int64
	err := row.Scan(&tag)
	if err == sql.ErrNoRows {
		return set, nil // empty
	} else if err != nil {
		return DuplicateSet{}, herror.Internal(err, "")
	}
	// get all others
	rows, err := s.db.Query("SELECT path, size, tag FROM duplicates WHERE tag = ? ORDER BY path", tag)
	if err != nil {
		return DuplicateSet{}, herror.Internal(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		var size int64
		if err := rows.Scan(&path, &size, &tag); err != nil {
			return DuplicateSet{}, herror.Internal(err, "")
		}
		set.Size = size
		set.Tag = tag
		set.Paths = append(set.Paths, path)
	}
	return set, nil
}

func (s *Session) LookupAll(path string, includeHidden bool) ([]DuplicateInfo, herror.Interface) {
	// we want to make sure the path ends in a '/', so we don't match files
	// that have the same prefix as the directory name
	if path[len(path)-1] != os.PathSeparator {
		path = path + string(os.PathSeparator)
	}
	var rows *sql.Rows
	var err error
	if includeHidden {
		rows, err = s.db.Query(`
		SELECT a.path, a.tag, COUNT(b.path)
		FROM duplicates a, duplicates b
		WHERE a.tag = b.tag AND SUBSTR(a.path, 1, ?) = ?
		GROUP BY a.path
		ORDER BY a.path
		`, len(path), path)
	} else {
		rows, err = s.db.Query(`
		SELECT a.path, a.tag, COUNT(b.path)
		FROM duplicates a, duplicates b
		WHERE a.tag = b.tag AND SUBSTR(a.path, 1, ?) = ? AND SUBSTR(a.path, ?) NOT LIKE '%/.%'
		GROUP BY a.path
		ORDER BY a.path
		`, len(path), path, len(path))
	}
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	defer rows.Close()
	var sets []DuplicateInfo
	for rows.Next() {
		var path string
		var tag int64
		var count int64
		if err := rows.Scan(&path, &tag, &count); err != nil {
			return nil, herror.Internal(err, "")
		}
		sets = append(sets, DuplicateInfo{Path: path, Tag: tag, Count: count})
	}
	return sets, nil
}

func (s *Session) Remove(path string) herror.Interface {
	_, err := s.db.Exec("DELETE FROM duplicates WHERE path = ?", path)
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

func (s *Session) RemoveAll(paths []string) herror.Interface {
	tx, err := s.db.Begin()
	if err != nil {
		return herror.Internal(err, "")
	}
	stmt, err := tx.Prepare("DELETE FROM duplicates WHERE path = ?")
	if err != nil {
		return herror.Internal(err, "")
	}
	defer stmt.Close()
	for _, path := range paths {
		if _, err := stmt.Exec(path); err != nil {
			return herror.Internal(err, "")
		}
	}
	err = tx.Commit()
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

func (s *Session) PruneSingletons() herror.Interface {
	_, err := s.db.Exec(`
	WITH singleton_tags AS
	(
		SELECT tag FROM duplicates GROUP BY tag
		HAVING COUNT(*) = 1
	)
	DELETE FROM duplicates
	where tag in singleton_tags
	`)
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}
