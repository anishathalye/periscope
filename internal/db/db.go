package db

import (
	"github.com/anishathalye/periscope/internal/herror"

	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"

	_ "github.com/mattn/go-sqlite3"
)

const versionKey = "version"
const version = 3

type FileInfo struct {
	Path      string
	Size      int64
	ShortHash []byte
	FullHash  []byte
}

type DuplicateSet []FileInfo

type fileInfosOrdering []FileInfo

func (a fileInfosOrdering) Len() int { return len(a) }
func (a fileInfosOrdering) Less(i, j int) bool {
	if a[i].Size > a[j].Size {
		return true
	} else if a[i].Size < a[j].Size {
		return false
	}
	return a[i].Path < a[j].Path
}
func (a fileInfosOrdering) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type DuplicateInfo struct {
	Path     string
	FullHash []byte
	Count    int64
}

type duplicateInfoByPath []DuplicateInfo

func (a duplicateInfoByPath) Len() int           { return len(a) }
func (a duplicateInfoByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }
func (a duplicateInfoByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type InfoSummary struct {
	Files     int64
	Unique    int64
	Duplicate int64
	Overhead  int64
}

// A database session, or a transaction.
//
// This is a sort of weird implementation, but it makes the
// interface/implementation convenient. The same type exposes a bunch of
// methods that operate on the database, and the object and methods are the
// same regardless of whether the operations are done within a transaction.
//
// Calling Begin() returns a Session that is a transaction, and calling
// Commit() on the resultant session (transaction) commits the transaction.
//
// The db field is non-nil for a new session. For an open transaction, db is
// nil and tx is non-nil. Once Commit() is called on the transaction, both the
// db and tx are nil (and any method calls on this object will fail).
type Session struct {
	db *sql.DB
	tx *sql.Tx
}

var inMemoryDbCtr int64 = 0

func NewInMemory() (*Session, herror.Interface) {
	// https://www.sqlite.org/inmemorydb.html#sharedmemdb
	//
	// We need distinct in-memory databases (for separate tests),
	// but each in-memory database should support multiple connections
	ctr := atomic.LoadInt64(&inMemoryDbCtr)
	atomic.StoreInt64(&inMemoryDbCtr, ctr+1)
	return New(fmt.Sprintf("file:memdb%d?mode=memory&cache=shared", ctr), true)
}

func New(dataSourceName string, debug bool) (*Session, herror.Interface) {
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
	// set up pragmas
	if debug {
		// good sanity check but slows things down, especially the gc in RemoveDir()
		db.Exec("PRAGMA foreign_keys = ON")
	} else {
		db.Exec("PRAGMA foreign_keys = OFF")
	}
	db.Exec("PRAGMA cache_size = -500000") // 500 MB

	s := &Session{db: db}
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

func (s *Session) checkVersion() herror.Interface {
	// ensure metadata table exists
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS meta
	(
		key   TEXT UNIQUE NOT NULL,
		value BLOB NOT NULL
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
	// only called in New, so db is non-null
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS directory
	(
		id     INTEGER PRIMARY KEY NOT NULL,
		name   TEXT NOT NULL,
		parent INTEGER NULL,
		FOREIGN KEY(parent) REFERENCES directory(id),
		UNIQUE(name, parent)
	)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS file_info
	(
		id         INTEGER PRIMARY KEY NOT NULL,
		directory  INTEGER NOT NULL,
		filename   TEXT NOT NULL,
		size       INTEGER NOT NULL,
		short_hash BLOB NULL,
		full_hash  BLOB NULL,
		FOREIGN KEY(directory) REFERENCES directory(id),
		UNIQUE(directory, filename)
	)
	`)
	return err
}

func (s *Session) Begin() (*Session, herror.Interface) {
	if s.tx != nil {
		return nil, herror.Internal(nil, "cannot Begin(): already in a transaction")
	}
	if s.db == nil {
		return nil, herror.Internal(nil, "cannot Begin(): finished transaction")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	return &Session{db: nil, tx: tx}, nil
}

func (s *Session) Commit() herror.Interface {
	if s.tx == nil {
		return herror.Internal(nil, "Commit(): not in a running transaction")
	}
	err := s.tx.Commit()
	if err != nil {
		return herror.Internal(err, "")
	}
	s.tx = nil
	return nil
}

func (s *Session) query(query string, args ...interface{}) (*sql.Rows, error) {
	if s.tx != nil {
		return s.tx.Query(query, args...)
	}
	if s.db == nil {
		return nil, herror.Internal(nil, "transaction is finished")
	}
	return s.db.Query(query, args...)
}

func (s *Session) queryRow(query string, args ...interface{}) (*sql.Row, herror.Interface) {
	if s.tx != nil {
		return s.tx.QueryRow(query, args...), nil
	}
	if s.db == nil {
		return nil, herror.Internal(nil, "transaction is finished")
	}
	return s.db.QueryRow(query, args...), nil
}

func (s *Session) exec(query string, args ...interface{}) (sql.Result, error) {
	if s.tx != nil {
		return s.tx.Exec(query, args...)
	}
	if s.db == nil {
		return nil, herror.Internal(nil, "transaction is finished")
	}
	return s.db.Exec(query, args...)
}

func (s *Session) pathToDirectoryId(path string, create bool) (int64, error) {
	if path == "" {
		return 0, errors.New("path is empty")
	}
	path = filepath.Clean(path) // remove extra "/" at the end, etc.
	var elems []string
	var base string
	for base != "/" {
		base = filepath.Base(path)
		elems = append(elems, base)
		path = filepath.Dir(path)
	}
	id := int64(-1)
	for i := len(elems) - 1; i >= 0; i-- {
		var row *sql.Row
		var err error
		if id == -1 {
			row, err = s.queryRow(`
			SELECT id
			FROM directory
			WHERE name = ?
				AND parent IS NULL
			`, elems[i])
		} else {
			row, err = s.queryRow(`
			SELECT id
			FROM directory
			WHERE name = ?
				AND parent = ?
			`, elems[i], id)
		}
		if err != nil {
			return 0, err
		}
		err = row.Scan(&id)
		if err == sql.ErrNoRows && create {
			// need to create it
			var result sql.Result
			if id == -1 {
				result, err = s.exec(`
				INSERT INTO directory (name, parent) VALUES (?, NULL)
				`, elems[i])
			} else {
				result, err = s.exec(`
				INSERT INTO directory (name, parent) VALUES (?, ?)
				`, elems[i], id)
			}
			if err != nil {
				return 0, err
			}
			id, err = result.LastInsertId()
			if err != nil {
				return 0, err
			}
		} else if err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Session) directoryIdToPath(id int64) (string, error) {
	rows, err := s.query(`
	WITH RECURSIVE sup_directory (id, name, parent, level) AS (
		SELECT id, name, parent, 1 FROM directory WHERE id = ?
		UNION ALL
		SELECT d.id, d.name, d.parent, level+1
		FROM directory d, sup_directory sd
		WHERE d.id = sd.parent
	)
	SELECT name, (SELECT max(level) FROM sup_directory) - level AS distance
	FROM sup_directory
	ORDER BY distance
	`, id)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var path string
	for rows.Next() {
		var name string
		var level int64
		if err = rows.Scan(&name, &level); err != nil {
			return "", err
		}
		if path == "" {
			path = name
		} else {
			path = filepath.Join(path, name)
		}
	}
	return path, nil
}

func (s *Session) Add(info FileInfo) herror.Interface {
	dirname := filepath.Dir(info.Path)
	filename := filepath.Base(info.Path)
	dirid, err := s.pathToDirectoryId(dirname, true)
	if err != nil {
		return herror.Internal(err, "")
	}
	if _, err := s.exec(`
	REPLACE INTO file_info (directory, filename, size, short_hash, full_hash)
	VALUES (?, ?, ?, ?, ?)
	`, dirid, filename, info.Size, info.ShortHash, info.FullHash); err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

// Returns all infos in the database (regardless of whether they have
// duplicates).
func (s *Session) AllInfosC() (<-chan FileInfo, herror.Interface) {
	rows, err := s.query(`
	SELECT directory, filename, size, short_hash, full_hash
	FROM file_info`)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	results := make(chan FileInfo)
	go func() {
		defer rows.Close()
		for rows.Next() {
			var dirid int64
			var filename string
			var info FileInfo
			if err := rows.Scan(&dirid, &filename, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
				// similar issue as below in AllDuplicatesC: how to report this?
				log.Printf("failure while scanning row: %s", err)
				continue
			}
			dirname, err := s.directoryIdToPath(dirid)
			if err != nil {
				log.Printf("failure while resolving directory name: %s", err)
				continue
			}
			info.Path = filepath.Join(dirname, filename)
			results <- info
		}
		close(results)
	}()
	return results, nil
}

func (s *Session) AllInfos() ([]FileInfo, herror.Interface) {
	var r []FileInfo
	c, err := s.AllInfosC()
	if err != nil {
		return nil, err
	}
	for i := range c {
		r = append(r, i)
	}
	sort.Sort(fileInfosOrdering(r))
	return r, nil
}

func (s *Session) CreateIndexes() herror.Interface {
	// ensuring that an index on full_hash exists makes a huge difference
	// in performance for commands like ls, because we use this for finding
	// duplicates
	_, err := s.exec("CREATE INDEX IF NOT EXISTS idx_hash ON file_info (full_hash)")
	if err != nil {
		return herror.Internal(err, "")
	}
	// makes a big difference when we are looking up by size (relevant when
	// scanning)
	_, err = s.exec("CREATE INDEX IF NOT EXISTS idx_size ON file_info (size)")
	if err != nil {
		return herror.Internal(err, "")
	}
	// for looking up files by directory/filename
	_, err = s.exec("CREATE INDEX IF NOT EXISTS idx_directory_filename ON file_info (directory, filename)")
	if err != nil {
		return herror.Internal(err, "")
	}
	// for recursive lookup
	_, err = s.exec("CREATE INDEX IF NOT EXISTS idx_name_parent ON directory (name, parent)")
	if err != nil {
		return herror.Internal(err, "")
	}
	// indexes on foreign keys
	_, err = s.exec("CREATE INDEX IF NOT EXISTS idx_directory ON file_info (directory)")
	if err != nil {
		return herror.Internal(err, "")
	}
	_, err = s.exec("CREATE INDEX IF NOT EXISTS idx_parent ON directory (parent)")
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

// Returns all known duplicates in the database.
//
// These are necessarily FileInfos with the FullHash field filled out. Each
// DuplicateSet that is returned always has > 1 element (i.e. it only includes
// duplicates, not infos where we happen to know the full hash).
//
// path is optional; if "", then all duplicates are returned, otherwise only
// ones with the given directory prefix
func (s *Session) AllDuplicatesC(path string) (<-chan DuplicateSet, herror.Interface) {
	results := make(chan DuplicateSet)
	dirid := int64(-1)
	var err error
	if path != "" {
		dirid, err = s.pathToDirectoryId(path, false)
		if err == sql.ErrNoRows {
			close(results)
			return results, nil
		} else if err != nil {
			return nil, herror.Internal(err, "")
		}
	}
	var rows *sql.Rows
	if dirid == -1 {
		rows, err = s.query(`
		SELECT directory, filename, size, short_hash, full_hash
		FROM file_info
		WHERE full_hash IS NOT NULL
		ORDER BY size DESC, full_hash`)
	} else {
		rows, err = s.query(`
		WITH dirs AS
		(
			WITH RECURSIVE sub_directory (id, parent) AS (
				SELECT id, parent FROM directory WHERE id = ?
				UNION ALL
				SELECT d.id, d.parent
				FROM directory d, sub_directory sd
				WHERE d.parent = sd.id
			)
			SELECT id FROM sub_directory
		),
		matching_hashes AS
		(
			SELECT full_hash FROM file_info WHERE directory IN dirs AND full_hash IS NOT NULL
		)
		SELECT directory, filename, size, short_hash, full_hash
		FROM file_info
		WHERE full_hash IN matching_hashes
		ORDER BY size DESC, full_hash`, dirid)
	}
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	go func() {
		defer rows.Close()
		var set DuplicateSet
		var prevHash []byte
		for rows.Next() {
			var dirid int64
			var filename string
			var info FileInfo
			if err := rows.Scan(&dirid, &filename, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
				// how should we handle this error that happens in its own goroutine?
				// give up on this row?
				log.Printf("failure while scanning row: %s", err)
				continue
			}
			dirname, err := s.directoryIdToPath(dirid)
			if err != nil {
				log.Printf("failure while resolving directory name: %s", err)
				continue
			}
			info.Path = filepath.Join(dirname, filename)
			if !bytes.Equal(info.FullHash, prevHash) {
				if len(set) > 1 {
					// note: set may have singletons, we don't remove info about files with single matches
					sort.Sort(fileInfosOrdering(set))
					results <- set
				}
				set = nil
			}
			prevHash = info.FullHash
			set = append(set, info)
		}
		// will usually be some infos left over, if the last file size/hash has duplicates
		if len(set) > 1 {
			sort.Sort(fileInfosOrdering(set))
			results <- set
		}
		close(results)
	}()
	return results, nil
}

func (s *Session) AllDuplicates(path string) ([]DuplicateSet, herror.Interface) {
	var r []DuplicateSet
	c, err := s.AllDuplicatesC(path)
	if err != nil {
		return nil, err
	}
	for d := range c {
		r = append(r, d)
	}
	return r, nil
}

func (s *Session) Summary() (InfoSummary, herror.Interface) {
	row, err := s.queryRow("SELECT COUNT(*) FROM file_info")
	if err != nil {
		return InfoSummary{}, err
	}
	var files int64
	if err := row.Scan(&files); err != nil {
		return InfoSummary{}, herror.Internal(err, "")
	}
	row, err = s.queryRow(`
	WITH sets AS
	(
		SELECT COUNT(*) AS cnt, size
		FROM file_info
		GROUP BY full_hash
		HAVING COUNT(full_hash) > 1
	)
	SELECT COUNT(*), SUM(cnt), SUM((cnt-1)*size) from sets
	`)
	if err != nil {
		return InfoSummary{}, err
	}
	var uniqueWithDuplicates int64
	var filesWithDuplicates, overhead sql.NullInt64
	if err := row.Scan(&uniqueWithDuplicates, &filesWithDuplicates, &overhead); err != nil {
		return InfoSummary{}, herror.Internal(err, "")
	}
	duplicate := filesWithDuplicates.Int64 - uniqueWithDuplicates
	return InfoSummary{
		Files:     files,
		Unique:    files - duplicate,
		Duplicate: duplicate,
		Overhead:  overhead.Int64,
	}, nil
}

// Returns info for everything matching the given file.
//
// Returns [] if there isn't a matching file in the database. If the file
// exists in the database, that file is returned first.
func (s *Session) Lookup(path string) (DuplicateSet, herror.Interface) {
	dirname := filepath.Dir(path)
	filename := filepath.Base(path)
	var set DuplicateSet
	dirid, err := s.pathToDirectoryId(dirname, false)
	if err == sql.ErrNoRows {
		return set, nil // directory not known => empty
	} else if err != nil {
		return nil, herror.Internal(err, "")
	}
	row, herr := s.queryRow(`
	SELECT id, size, short_hash, full_hash
	FROM file_info
	WHERE directory = ? AND filename = ?
	`, dirid, filename)
	if herr != nil {
		return nil, herr
	}
	var id int
	var info FileInfo
	err = row.Scan(&id, &info.Size, &info.ShortHash, &info.FullHash)
	if err == sql.ErrNoRows {
		return set, nil // empty
	} else if err != nil {
		return nil, herror.Internal(err, "")
	}
	info.Path = filepath.Join(dirname, filename)
	if info.FullHash == nil {
		// no known duplicates
		set = append(set, info)
		return set, nil
	}
	// get all others
	rows, err := s.query(`
	SELECT directory, filename, size, short_hash, full_hash
	FROM file_info
	WHERE full_hash = ? AND id != ?`, info.FullHash, id)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var info FileInfo
		if err := rows.Scan(&dirid, &filename, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
			return nil, herror.Internal(err, "")
		}
		dirname, err := s.directoryIdToPath(dirid)
		if err != nil {
			return nil, herror.Internal(err, "")
		}
		info.Path = filepath.Join(dirname, filename)
		set = append(set, info)
	}
	sort.Sort(fileInfosOrdering(set))
	set = append(DuplicateSet{info}, set...) // so the given info is first
	return set, nil
}

// Returns all the infos with the given size.
//
// This includes all infos, even ones where the short hash or full hash is not known.
func (s *Session) InfosBySize(size int64) ([]FileInfo, herror.Interface) {
	rows, err := s.query(`
	SELECT directory, filename, size, short_hash, full_hash
	FROM file_info
	WHERE size = ?
	`, size)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	defer rows.Close()
	var results []FileInfo
	for rows.Next() {
		var dirid int64
		var filename string
		var info FileInfo
		if err := rows.Scan(&dirid, &filename, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
			return nil, herror.Internal(err, "")
		}
		dirname, err := s.directoryIdToPath(dirid)
		if err != nil {
			return nil, herror.Internal(err, "")
		}
		info.Path = filepath.Join(dirname, filename)
		results = append(results, info)
	}
	return results, nil
}

// Returns all duplicate sets (size > 1) where at least one file is contained under the given path.
func (s *Session) LookupAllC(path string, includeHidden bool) (<-chan DuplicateInfo, herror.Interface) {
	results := make(chan DuplicateInfo)
	dirid, err := s.pathToDirectoryId(path, false)
	if err == sql.ErrNoRows {
		close(results)
		return results, nil
	} else if err != nil {
		return nil, herror.Internal(err, "")
	}
	var rows *sql.Rows
	if includeHidden {
		rows, err = s.query(`
		WITH dirs AS
		(
			WITH RECURSIVE sub_directory (id, parent) AS (
				SELECT id, parent FROM directory WHERE id = ?
				UNION ALL
				SELECT d.id, d.parent
				FROM directory d, sub_directory sd
				WHERE d.parent = sd.id
			)
			SELECT id FROM sub_directory
		)
		SELECT a.directory, a.filename, a.full_hash, COUNT(b.id)
		FROM file_info a, file_info b
		WHERE a.full_hash IS NOT NULL
			AND a.full_hash = b.full_hash
			AND a.directory IN dirs
		GROUP BY a.directory, a.filename
		`, dirid)
	} else {
		rows, err = s.query(`
		WITH dirs AS
		(
			WITH RECURSIVE sub_directory (id, parent) AS (
				SELECT id, parent FROM directory WHERE id = ?
				UNION ALL
				SELECT d.id, d.parent
				FROM directory d, sub_directory sd
				WHERE d.parent = sd.id
					AND SUBSTR(d.name, 1, 1) != '.'
			)
			SELECT id FROM sub_directory
		)
		SELECT a.directory, a.filename, a.full_hash, COUNT(b.id)
		FROM file_info a, file_info b
		WHERE a.full_hash IS NOT NULL
			AND a.full_hash = b.full_hash
			AND a.directory IN dirs
			AND SUBSTR(a.filename, 1, 1) != '.'
		GROUP BY a.directory, a.filename
		`, dirid)
	}
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	go func() {
		defer rows.Close()
		for rows.Next() {
			var dirid int64
			var filename string
			var fullHash []byte
			var count int64
			if err := rows.Scan(&dirid, &filename, &fullHash, &count); err != nil {
				log.Printf("failure while scanning row: %s", err)
				continue
			}
			dirname, err := s.directoryIdToPath(dirid)
			if err != nil {
				log.Printf("failure while resolving directory name: %s", err)
				continue
			}
			path := filepath.Join(dirname, filename)
			if count > 1 {
				results <- DuplicateInfo{Path: path, FullHash: fullHash, Count: count}
			}
		}
		close(results)
	}()
	return results, nil
}

func (s *Session) LookupAll(path string, includeHidden bool) ([]DuplicateInfo, herror.Interface) {
	var r []DuplicateInfo
	c, err := s.LookupAllC(path, includeHidden)
	if err != nil {
		return nil, err
	}
	for i := range c {
		r = append(r, i)
	}
	sort.Sort(duplicateInfoByPath(r))
	return r, nil
}

// Deletes a file with the given path from the database.
func (s *Session) Remove(path string) herror.Interface {
	dirname := filepath.Dir(path)
	filename := filepath.Base(path)
	dirid, err := s.pathToDirectoryId(dirname, true)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return herror.Internal(err, "")
	}
	_, err = s.exec(`
	DELETE FROM file_info
	WHERE directory = ? AND filename = ?`, dirid, filename)
	if err != nil {
		return herror.Internal(err, "")
	}
	// don't bother to delete orphaned directories here
	return nil
}

// Deletes all files matching the given directory prefix from the database,
// with sizes in the specified range.
//
// A max size of 0 is interpreted as infinity. This does not just match based
// on prefix, it interprets the prefix as a directory, and only deletes files
// under the given directory. This means that it won't accidentally match file
// names (or other directory names) where the prefix is common, e.g. deleting
// "/a" won't delete file "/aa" or contents under a directory "/aa".
func (s *Session) RemoveDir(dir string, min, max int64) herror.Interface {
	if max <= 0 {
		max = math.MaxInt64
	}
	dirid, err := s.pathToDirectoryId(dir, false)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return herror.Internal(err, "")
	}
	if min == 0 && max == math.MaxInt64 {
		// more efficient query
		_, err = s.exec(`
		WITH dirs AS
		(
			WITH RECURSIVE sub_directory (id, parent) AS (
				SELECT id, parent FROM directory WHERE id = ?
				UNION ALL
				SELECT d.id, d.parent
				FROM directory d, sub_directory sd
				WHERE d.parent = sd.id
			)
			SELECT id FROM sub_directory
		)
		DELETE FROM file_info
		WHERE directory IN dirs
		`, dirid)
	} else {
		_, err = s.exec(`
		WITH dirs AS
		(
			WITH RECURSIVE sub_directory (id, parent) AS (
				SELECT id, parent FROM directory WHERE id = ?
				UNION ALL
				SELECT d.id, d.parent
				FROM directory d, sub_directory sd
				WHERE d.parent = sd.id
			)
			SELECT id FROM sub_directory
		)
		DELETE FROM file_info
		WHERE directory IN dirs
			AND size > ?
			AND size <= ?
		`, dirid, min, max)
	}
	if err != nil {
		return herror.Internal(err, "")
	}
	// delete orphaned directories
	_, err = s.exec(`
	WITH reachable AS
	(
		WITH RECURSIVE sub_directory (id, parent) AS (
			SELECT id, parent FROM directory WHERE id IN (SELECT DISTINCT directory FROM file_info)
			UNION ALL
			SELECT d.id, d.parent
			FROM directory d, sub_directory sd
			WHERE d.id = sd.parent
		)
		SELECT DISTINCT id
		FROM sub_directory
	)
	DELETE FROM directory
	WHERE id NOT IN reachable`)
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}
