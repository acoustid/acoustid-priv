package priv

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"log"
)

const NumIndexSegments = 16
const ValuesPerSegment = 128
const QueryBits = 26

type SearchOptions struct {
	Stream bool
}

type SearchResults struct {
	Results []SearchResult
}

type SearchResult struct {
	ID       string
	Metadata Metadata
	Score    int
}

type Metadata map[string]string

type Catalog interface {
	Name() string

	Exists() (bool, error)
	CreateCatalog() error
	DeleteCatalog() error

	NewTrackID() string

	CreateTrack(id string, fp *chromaprint.Fingerprint, meta Metadata) error
	DeleteTrack(id string) error

	Search(query *chromaprint.Fingerprint, opts *SearchOptions) (*SearchResults, error)
}

type CatalogImpl struct {
	db   *sql.DB
	repo *RepositoryImpl
	name string
	id   int
}

func (c *CatalogImpl) Name() string {
	return c.name
}

func (c *CatalogImpl) checkCatalog(tx *sql.Tx) (bool, error) {
	if c.id != 0 {
		return true, nil
	}

	row := tx.QueryRow("SELECT id FROM catalog WHERE account_id = $1 AND name = $2", c.repo.account.id, c.name)
	err := row.Scan(&c.id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.WithMessage(err, "failed to get catalog")
	}

	if c.id != 0 {
		return true, nil
	}

	return false, nil
}

func (c *CatalogImpl) Exists() (bool, error) {
	tx, err := c.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return false, errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	exists, err := c.checkCatalog(tx)
	if exists {
		return true, nil
	}

	return false, nil
}

func (c *CatalogImpl) CreateCatalog() error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	exists, err := c.checkCatalog(tx)
	if exists {
		return nil
	}

	row := tx.QueryRow("INSERT INTO catalog (account_id, name) VALUES ($1, $2) RETURNING id", c.repo.account.id, c.name)
	var id int
	err = row.Scan(&id)
	if err != nil {
		return errors.WithMessage(err, "failed to create catalog")
	}

	_, err = tx.Exec(fmt.Sprintf("CREATE TABLE track_%d (LIKE track_tpl INCLUDING ALL)", id))
	if err != nil {
		return errors.WithMessage(err, "failed to create track table")
	}

	for i := 0; i < NumIndexSegments; i++ {
		_, err = tx.Exec(fmt.Sprintf("CREATE TABLE track_index_%d_%d (LIKE track_index_tpl INCLUDING ALL)", id, i))
		if err != nil {
			return errors.WithMessage(err, "failed to create track index table")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.WithMessage(err, "commit failed")
	}

	c.id = id
	log.Printf("Created catalog name=%v account_id=%v", c.name, c.repo.account.id)
	return nil
}

func (c *CatalogImpl) DeleteCatalog() error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	row := tx.QueryRow("DELETE FROM catalog WHERE account_id = $1 AND name = $2 RETURNING id", c.repo.account.id, c.name)
	var id int
	err = row.Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.WithMessage(err, "failed to delete catalog table")
	}

	_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS track_%d", id))
	if err != nil {
		return errors.WithMessage(err, "failed to create track table")
	}

	for i := 0; i < NumIndexSegments; i++ {
		_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS track_index_%d_%d", id, i))
		if err != nil {
			return errors.WithMessage(err, "failed to create track index table")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.WithMessage(err, "commit failed")
	}

	c.id = 0
	log.Printf("Deleted catalog name=%v account_id=%v", c.name, c.repo.account.id)
	return nil
}

func (c *CatalogImpl) NewTrackID() string {
	return uuid.NewV4().String()
}

func (c *CatalogImpl) CreateTrack(externalID string, fingerprint *chromaprint.Fingerprint, metadata Metadata) error {
	err := c.CreateCatalog()
	if err != nil {
		return err
	}

	tx, err := c.db.Begin()
	if err != nil {
		return errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	deleted, err := c.deleteTrack(tx, externalID)
	if err != nil {
		return err
	}

	fingerprintBytes := chromaprint.CompressFingerprint(*fingerprint)
	fingerprintSHA1 := sha1.Sum(fingerprintBytes)

	var metadataBytes *[]byte = nil
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return errors.WithMessage(err, "failed to encode metadata")
		}
		metadataBytes = &data
	}

	query := fmt.Sprintf("INSERT INTO track_%d (external_id, fingerprint, fingerprint_sha1, metadata) VALUES ($1, $2, $3, $4) RETURNING id", c.id)
	row := tx.QueryRow(query, externalID, fingerprintBytes, fingerprintSHA1[:], metadataBytes)
	var internalID int
	err = row.Scan(&internalID)
	if err != nil {
		return errors.WithMessage(err, "failed to insert track")
	}

	segment := 0
	values := ExtractQuery(fingerprint, QueryBits)
	for i := 0; i < len(fingerprint.Hashes); i += ValuesPerSegment {
		n := ValuesPerSegment
		if len(fingerprint.Hashes)-i < n {
			n = len(fingerprint.Hashes) - i
		}
		query := fmt.Sprintf("INSERT INTO track_index_%d_%d (track_id, segment, values) VALUES ($1, $2, $3)", c.id, segment%NumIndexSegments)
		_, err = tx.Exec(query, internalID, segment, pq.Array(values[i:i+n]))
		if err != nil {
			return errors.WithMessage(err, "failed to insert track index")
		}
		segment += 1
	}

	err = tx.Commit()
	if err != nil {
		return errors.WithMessage(err, "commit failed")
	}

	if deleted {
		log.Printf("Updated track id=%v catalog=%s account_id=%v", externalID, c.name, c.repo.account.id)
	} else {
		log.Printf("Inserted track id=%v catalog=%s account_id=%v", externalID, c.name, c.repo.account.id)
	}

	return nil
}

func (c *CatalogImpl) deleteTrack(tx *sql.Tx, externalID string) (bool, error) {
	row := tx.QueryRow(fmt.Sprintf("DELETE FROM track_%d WHERE external_id = $1 RETURNING id", c.id), externalID)
	var internalID int
	err := row.Scan(&internalID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.WithMessage(err, "failed to delete track")
	}

	for i := 0; i < NumIndexSegments; i++ {
		query := fmt.Sprintf("DELETE FROM track_index_%d_%d WHERE track_id = $1", c.id, i)
		_, err = tx.Exec(query, internalID)
		if err != nil {
			return false, errors.WithMessage(err, "failed to delete track index")
		}
	}

	return true, nil
}

func (c *CatalogImpl) DeleteTrack(externalID string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	exists, err := c.checkCatalog(tx)
	if !exists {
		return nil
	}

	deleted, err := c.deleteTrack(tx, externalID)
	if err != nil {
		return err
	}
	if !deleted {
		return nil
	}

	err = tx.Commit()
	if err != nil {
		return errors.WithMessage(err, "commit failed")
	}

	log.Printf("Deleted track id=%v catalog=%s account_id=%v", externalID, c.name, c.repo.account.id)
	return nil

}

func (c *CatalogImpl) Search(fingerprint *chromaprint.Fingerprint, opts *SearchOptions) (*SearchResults, error) {
	tx, err := c.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	results := &SearchResults{}

	exists, err := c.checkCatalog(tx)
	if !exists {
		return results, nil
	}

	hits := make(map[int]int)
	values := ExtractQuery(fingerprint, QueryBits)
	for i := 0; i < NumIndexSegments; i++ {
		queryTpl := "SELECT track_id, icount(values & query) " +
			"FROM track_index_%d_%d, (SELECT $1::int[] AS query) q " +
			"WHERE values && query"
		query := fmt.Sprintf(queryTpl, c.id, i%NumIndexSegments)
		rows, err := tx.Query(query, pq.Array(values))
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var trackID, trackHits int
			err = rows.Scan(&trackID, &trackHits)
			if err != nil {
				rows.Close()
				return nil, err
			}
			hits[trackID] += trackHits
		}
		rows.Close()
	}

	results.Results = make([]SearchResult, 0, len(hits))

	queryTpl := "SELECT id, external_id, metadata FROM track_%d WHERE id = any($1::int[])"
	query := fmt.Sprintf(queryTpl, c.id)
	trackIDs := make([]int, 0, len(hits))
	for trackID := range hits {
		trackIDs = append(trackIDs, trackID)
	}
	rows, err := tx.Query(query, pq.Array(trackIDs))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var trackID int
		var externalTrackID string
		var metadata *Metadata
		err = rows.Scan(&trackID, &externalTrackID, &metadata)
		if err != nil {
			return nil, err
		}
		result := SearchResult{
			ID:    externalTrackID,
			Score: hits[trackID],
		}
		if metadata != nil {
			result.Metadata = *metadata
		}
		results.Results = append(results.Results, result)
	}

	return results, nil
}
