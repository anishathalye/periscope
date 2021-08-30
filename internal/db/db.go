package db

import (
	"github.com/anishathalye/periscope/internal/herror"

	"bytes"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type FileInfo struct {
	Path      string
	Size      int64
	ShortHash []byte
	FullHash  []byte
}

type DuplicateSet []FileInfo

type DuplicateInfo struct {
	Path     string
	FullHash []byte
	Count    int64
}

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

const versionKey = "version"
const version = 2

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
	CREATE TABLE IF NOT EXISTS file_info
	(
		id         INTEGER PRIMARY KEY NOT NULL,
		path       TEXT UNIQUE NOT NULL,
		size       INTEGER NOT NULL,
		short_hash BLOB NULL,
		full_hash  BLOB NULL
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

func (s *Session) Add(info FileInfo) herror.Interface {
	if _, err := s.exec(`
	REPLACE INTO file_info (path, size, short_hash, full_hash)
	VALUES (?, ?, ?, ?)`, info.Path, info.Size, info.ShortHash, info.FullHash); err != nil {
		return herror.Internal(err, "")
	}
	return nil
}

// Returns all infos in the database (regardless of whether they have
// duplicates).
func (s *Session) AllInfosC() (<-chan FileInfo, herror.Interface) {
	rows, err := s.query(`
	SELECT path, size, short_hash, full_hash
	FROM file_info
	ORDER BY size DESC, path`)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	results := make(chan FileInfo)
	go func() {
		defer rows.Close()
		for rows.Next() {
			var info FileInfo
			if err := rows.Scan(&info.Path, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
				// similar issue as below in AllDuplicatesC: how to report this?
				log.Printf("failure while scanning row: %s", err)
				continue
			}
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
	return nil
}

// Returns all known duplicates in the database.
//
// These are necessarily FileInfos with the FullHash field filled out. Each
// DuplicateSet that is returned always has > 1 element (i.e. it only includes
// duplicates, not infos where we happen to know the full hash).
func (s *Session) AllDuplicatesC() (<-chan DuplicateSet, herror.Interface) {
	results := make(chan DuplicateSet)
	rows, err := s.query(`
	SELECT path, size, short_hash, full_hash
	FROM file_info
	WHERE full_hash IS NOT NULL
	ORDER BY size DESC, full_hash, path`)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	go func() {
		defer rows.Close()
		var set DuplicateSet
		var prevHash []byte
		for rows.Next() {
			var info FileInfo
			if err := rows.Scan(&info.Path, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
				// how should we handle this error that happens in its own goroutine?
				// give up on this row?
				log.Printf("failure while scanning row: %s", err)
				continue
			}
			if !bytes.Equal(info.FullHash, prevHash) {
				if len(set) > 1 {
					// note: set may have singletons, we don't remove info about files with single matches
					results <- set
				}
				set = nil
			}
			prevHash = info.FullHash
			set = append(set, info)
		}
		// will usually be some infos left over, if the last file size/hash has duplicates
		if len(set) > 1 {
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
	var set DuplicateSet
	row, herr := s.queryRow("SELECT path, size, short_hash, full_hash FROM file_info WHERE path = ?", path)
	if herr != nil {
		return nil, herr
	}
	var info FileInfo
	err := row.Scan(&info.Path, &info.Size, &info.ShortHash, &info.FullHash)
	if err == sql.ErrNoRows {
		return set, nil // empty
	} else if err != nil {
		return nil, herror.Internal(err, "")
	}
	set = append(set, info)
	if info.FullHash == nil {
		// no known duplicates
		return set, nil
	}
	// get all others
	rows, err := s.query(`
	SELECT path, size, short_hash, full_hash
	FROM file_info
	WHERE full_hash = ? AND path != ?
	ORDER BY path`, info.FullHash, path)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var info FileInfo
		if err := rows.Scan(&info.Path, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
			return nil, herror.Internal(err, "")
		}
		set = append(set, info)
	}
	return set, nil
}

// Returns all the infos with the given size.
//
// This includes all infos, even ones where the short hash or full hash is not known.
func (s *Session) InfosBySize(size int64) ([]FileInfo, herror.Interface) {
	rows, err := s.query("SELECT path, size, short_hash, full_hash FROM file_info WHERE size = ?", size)
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	defer rows.Close()
	var results []FileInfo
	for rows.Next() {
		var info FileInfo
		if err := rows.Scan(&info.Path, &info.Size, &info.ShortHash, &info.FullHash); err != nil {
			return nil, herror.Internal(err, "")
		}
		results = append(results, info)
	}
	return results, nil
}

// Returns all duplicate sets (size > 1) where at least one file is contained under the given path.
func (s *Session) LookupAllC(path string, includeHidden bool) (<-chan DuplicateInfo, herror.Interface) {
	// we want to make sure the path ends in a '/', so we don't match files
	// that have the same prefix as the directory name
	if path[len(path)-1] != os.PathSeparator {
		path = path + string(os.PathSeparator)
	}
	var rows *sql.Rows
	var err error
	if includeHidden {
		rows, err = s.query(`
		SELECT a.path, a.full_hash, COUNT(b.path)
		FROM file_info a, file_info b
		WHERE a.full_hash IS NOT NULL
			AND a.full_hash = b.full_hash
			AND SUBSTR(a.path, 1, ?) = ?
		GROUP BY a.path
		ORDER BY a.path
		`, len(path), path)
	} else {
		rows, err = s.query(`
		SELECT a.path, a.full_hash, COUNT(b.path)
		FROM file_info a, file_info b
		WHERE a.full_hash IS NOT NULL
			AND a.full_hash = b.full_hash
			AND SUBSTR(a.path, 1, ?) = ?
			AND SUBSTR(a.path, ?) NOT LIKE '%/.%'
		GROUP BY a.path
		ORDER BY a.path
		`, len(path), path, len(path))
	}
	if err != nil {
		return nil, herror.Internal(err, "")
	}
	results := make(chan DuplicateInfo)
	go func() {
		defer rows.Close()
		for rows.Next() {
			var path string
			var fullHash []byte
			var count int64
			if err := rows.Scan(&path, &fullHash, &count); err != nil {
				log.Printf("failure while scanning row: %s", err)
				continue
			}
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
	return r, nil
}

// Deletes a file with the given path from the database.
func (s *Session) Remove(path string) herror.Interface {
	_, err := s.exec("DELETE FROM file_info WHERE path = ?", path)
	if err != nil {
		return herror.Internal(err, "")
	}
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
	if dir[len(dir)-1] != os.PathSeparator {
		dir = dir + string(os.PathSeparator)
	}
	if max <= 0 {
		max = math.MaxInt64
	}
	_, err := s.exec(`
	DELETE FROM file_info
	WHERE SUBSTR(path, 1, ?) = ?
		AND size > ?
		AND size <= ?
	`, len(dir), dir, min, max)
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}
