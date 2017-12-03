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
	"sort"
	"sync"
	"time"
)

const SearchConcurrency = 8
const NumIndexSegments = 16
const ValuesPerSegment = 128

type SearchOptions struct {
	Stream bool
}

type SearchResults struct {
	Results []SearchResult
}

type SearchResult struct {
	ID       string
	Metadata Metadata
	Match    *MatchResult
}

type Metadata map[string]string

type Catalog interface {
	Name() string

	Exists() (bool, error)
	CreateCatalog() error
	DeleteCatalog() error

	NewTrackID() string

	GetTrack(id string) (*SearchResults, error)
	CreateTrack(id string, fp *chromaprint.Fingerprint, meta Metadata, allowDuplicate bool) (bool, error)
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

func (c *CatalogImpl) findTrackByFingerprintSHA1(tx *sql.Tx, fingerprintSHA1 []byte) (bool, error) {
	query := fmt.Sprintf("SELECT count(*) FROM track_%d WHERE fingerprint_sha1 = $1", c.id)
	row := tx.QueryRow(query, fingerprintSHA1)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *CatalogImpl) CreateTrack(externalID string, fingerprint *chromaprint.Fingerprint, metadata Metadata, allowDuplicate bool) (bool, error) {
	err := c.CreateCatalog()
	if err != nil {
		return false, err
	}

	tx, err := c.db.Begin()
	if err != nil {
		return false, errors.WithMessage(err, "failed to open transaction")
	}
	defer tx.Rollback()

	deleted, err := c.deleteTrack(tx, externalID)
	if err != nil {
		return false, err
	}

	fingerprintBytes := chromaprint.CompressFingerprint(*fingerprint)
	fingerprintSHA1 := sha1.Sum(fingerprintBytes)

	if !allowDuplicate {
		exists, err := c.findTrackByFingerprintSHA1(tx, fingerprintSHA1[:])
		if err != nil {
			return false, err
		}
		if exists {
			return false, nil
		}
	}

	var metadataBytes *[]byte = nil
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return false, errors.WithMessage(err, "failed to encode metadata")
		}
		metadataBytes = &data
	}

	query := fmt.Sprintf("INSERT INTO track_%d (external_id, fingerprint, fingerprint_sha1, metadata) VALUES ($1, $2, $3, $4) RETURNING id", c.id)
	row := tx.QueryRow(query, externalID, fingerprintBytes, fingerprintSHA1[:], metadataBytes)
	var internalID int
	err = row.Scan(&internalID)
	if err != nil {
		return false, errors.WithMessage(err, "failed to insert track")
	}

	segment := 0
	values := ExtractQuery(fingerprint)
	for i := 0; i < len(fingerprint.Hashes); i += ValuesPerSegment {
		n := ValuesPerSegment
		if len(fingerprint.Hashes)-i < n {
			n = len(fingerprint.Hashes) - i
		}
		query := fmt.Sprintf("INSERT INTO track_index_%d_%d (track_id, segment, values) VALUES ($1, $2, $3)", c.id, segment%NumIndexSegments)
		_, err = tx.Exec(query, internalID, segment, pq.Array(values[i:i+n]))
		if err != nil {
			return false, errors.WithMessage(err, "failed to insert track index")
		}
		segment += 1
	}

	err = tx.Commit()
	if err != nil {
		return false, errors.WithMessage(err, "commit failed")
	}

	if deleted {
		log.Printf("Updated track id=%v catalog=%s account_id=%v", externalID, c.name, c.repo.account.id)
	} else {
		log.Printf("Inserted track id=%v catalog=%s account_id=%v", externalID, c.name, c.repo.account.id)
	}
	return true, nil
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

func (c *CatalogImpl) searchFingerprintIndexSegment(values []int32, segment int) (map[int]int, error) {
	queryTpl := "SELECT track_id, icount(values & query) " +
		"FROM track_index_%d_%d, (SELECT $1::int[] AS query) q " +
		"WHERE values && query"
	query := fmt.Sprintf(queryTpl, c.id, segment%NumIndexSegments)
	rows, err := c.db.Query(query, pq.Array(values))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make(map[int]int)
	for rows.Next() {
		var trackID, count int
		err = rows.Scan(&trackID, &count)
		if err != nil {
			return nil, err
		}
		hits[trackID] += count
	}

	return hits, nil
}

func (c *CatalogImpl) searchFingerprintIndex(values []int32, stream bool) (map[int]int, error) {
	var segmentValues [NumIndexSegments][]int32
	var segmentHits [NumIndexSegments]map[int]int
	var segmentErrs [NumIndexSegments]error

	if stream {
		for segment := 0; segment < NumIndexSegments; segment++ {
			segmentValues[segment] = values
		}
	} else {
		if len(values) < ValuesPerSegment {
			segmentValues[0] = values
		} else {
			segmentValues[0] = values[:ValuesPerSegment]
			if len(values) < ValuesPerSegment*2 {
				segmentValues[1] = values[ValuesPerSegment:]
			} else {
				segmentValues[1] = values[ValuesPerSegment : ValuesPerSegment*2]
			}
		}
	}

	var wg sync.WaitGroup
	for chunk := 0; chunk < SearchConcurrency; chunk++ {
		wg.Add(1)
		go func(chunk int) {
			defer wg.Done()
			for segment := 0; segment < NumIndexSegments; segment++ {
				if segment%SearchConcurrency == chunk {
					values := segmentValues[segment]
					if len(values) != 0 {
						hits, err := c.searchFingerprintIndexSegment(values, segment)
						segmentHits[segment] = hits
						segmentErrs[segment] = err
					}
				}
			}
		}(chunk)
	}
	wg.Wait()

	hits := make(map[int]int)
	for segment := 0; segment < NumIndexSegments; segment++ {
		err := segmentErrs[segment]
		if err != nil {
			return nil, err
		}
		for trackID, count := range segmentHits[segment] {
			hits[trackID] += count
		}
	}

	return hits, nil
}

func (c *CatalogImpl) matchFingerprint(trackID int, queryFP *chromaprint.Fingerprint) (*MatchResult, error) {
	queryTpl := "SELECT fingerprint FROM track_%d WHERE id = $1"
	query := fmt.Sprintf(queryTpl, c.id)
	row := c.db.QueryRow(query, trackID)
	var data []byte
	err := row.Scan(&data)
	if err != nil {
		return nil, err
	}
	masterFP, err := chromaprint.ParseFingerprint(data)
	if err != nil {
		return nil, err
	}
	return MatchFingerprints(masterFP, queryFP)
}

func (c *CatalogImpl) Search(queryFP *chromaprint.Fingerprint, opts *SearchOptions) (*SearchResults, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	started := time.Now()

	searchType := "normal"
	if opts.Stream {
		searchType = "stream"
	}
	searchCount.WithLabelValues(searchType).Inc()

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

	values := ExtractQuery(queryFP)

	hits, err := c.searchFingerprintIndex(values, opts.Stream)
	if err != nil {
		return nil, errors.WithMessage(err, "index search failed")
	}

	maxCount := 0
	for _, count := range hits {
		if count > maxCount {
			maxCount = count
		}
	}
	countThreshold := maxCount / 10
	if countThreshold < 2 {
		countThreshold = 2
	}

	type TopHit struct {
		TrackID int
		Count   int
	}

	topHits := make([]TopHit, 0, len(hits))
	for trackID, count := range hits {
		if count >= countThreshold {
			topHits = append(topHits, TopHit{trackID, count})
		}
	}
	sort.Slice(topHits, func(i, j int) bool { return topHits[i].Count < topHits[j].Count })

	matches := make(map[int]*MatchResult)
	matchingTrackIDs := make([]int, 0, len(topHits))

	for _, hit := range topHits {
		_, exists := matches[hit.TrackID]
		if !exists {
			match, err := c.matchFingerprint(hit.TrackID, queryFP)
			if err != nil {
				return nil, errors.WithMessage(err, "matching failed")
			}
			if !match.Empty() {
				matches[hit.TrackID] = match
				matchingTrackIDs = append(matchingTrackIDs, hit.TrackID)
			}
		}
	}

	queryTpl := "SELECT id, external_id, metadata FROM track_%d WHERE id = any($1::int[])"
	query := fmt.Sprintf(queryTpl, c.id)
	rows, err := tx.Query(query, pq.Array(matchingTrackIDs))
	if err != nil {
		return nil, err
	}
	results.Results = make([]SearchResult, 0, len(topHits))
	for rows.Next() {
		var trackID int
		var externalTrackID string
		var metadataBytes json.RawMessage
		err = rows.Scan(&trackID, &externalTrackID, &metadataBytes)
		if err != nil {
			return nil, err
		}
		result := SearchResult{
			ID:    externalTrackID,
			Match: matches[trackID],
		}
		if metadataBytes != nil {
			err = json.Unmarshal(metadataBytes, &result.Metadata)
			if err != nil {
				return nil, errors.WithMessage(err, "metadata parsing failed")
			}
		}
		results.Results = append(results.Results, result)
	}

	searchDuration.WithLabelValues(searchType).Observe(time.Since(started).Seconds())

	return results, nil
}

func (c *CatalogImpl) GetTrack(externalID string) (*SearchResults, error) {
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

	query := fmt.Sprintf("SELECT metadata FROM track_%d WHERE external_id = $1", c.id)
	row := tx.QueryRow(query, externalID)
	var metadataBytes json.RawMessage
	err = row.Scan(&metadataBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return results, nil
		}
		return nil, errors.WithMessage(err, "failed to fetch track")
	}

	result := SearchResult{ID: externalID}
	if metadataBytes != nil {
		err = json.Unmarshal(metadataBytes, &result.Metadata)
		if err != nil {
			return nil, errors.WithMessage(err, "metadata parsing failed")
		}
	}
	results.Results = append(results.Results, result)
	return results, nil
}
