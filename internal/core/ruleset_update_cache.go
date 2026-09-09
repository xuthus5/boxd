package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"go.etcd.io/bbolt"
)

const ruleSetCacheFormatVersion = 2

type savedRuleSetBinary struct {
	Content     []byte
	LastUpdated time.Time
	LastEtag    string
	URLHash     []byte
}

func (s *savedRuleSetBinary) MarshalBinary() ([]byte, error) {
	if len(s.Content) > maxRuleSetBodyBytes {
		return nil, ErrRuleSetContentTooLarge
	}
	if len(s.LastEtag) > maxRuleSetEtagBytes {
		return nil, errors.New("rule-set etag is too large")
	}
	if len(s.URLHash) > sha256.Size {
		return nil, errors.New("rule-set URL hash is too large")
	}
	return (&adapter.SavedBinary{
		Content: s.Content, LastUpdated: s.LastUpdated, LastEtag: s.LastEtag, URLHash: s.URLHash,
	}).MarshalBinary()
}

func (s *savedRuleSetBinary) UnmarshalBinary(data []byte) error {
	reader := bytes.NewReader(data)
	version, err := reader.ReadByte()
	if err != nil {
		return err
	}
	if version != ruleSetCacheFormatVersion {
		return fmt.Errorf("unsupported rule-set cache version %d", version)
	}
	s.Content, err = readRuleSetCacheBytes(reader, maxRuleSetBodyBytes, ErrRuleSetContentTooLarge)
	if err != nil {
		return err
	}
	var updated int64
	if err = binary.Read(reader, binary.BigEndian, &updated); err != nil {
		return err
	}
	s.LastUpdated = time.Unix(updated, 0)
	etag, err := readRuleSetCacheBytes(reader, maxRuleSetEtagBytes, errors.New("rule-set etag is too large"))
	if err != nil {
		return err
	}
	s.LastEtag = string(etag)
	s.URLHash, err = readRuleSetCacheBytes(reader, sha256.Size, errors.New("rule-set URL hash is too large"))
	return err
}

func readRuleSetCacheBytes(reader *bytes.Reader, limit uint64, oversized error) ([]byte, error) {
	length, err := readUvarint(reader)
	if err != nil {
		return nil, err
	}
	if length > limit {
		return nil, oversized
	}
	if length > uint64(reader.Len()) {
		return nil, io.ErrUnexpectedEOF
	}
	result := make([]byte, length)
	_, err = io.ReadFull(reader, result)
	return result, err
}

func ruleSetURLHash(url string) []byte {
	sum := sha256.Sum256([]byte(url))
	return sum[:]
}

func loadRuleSetCacheForURL(db *bbolt.DB, tag, url string) *savedRuleSetBinary {
	saved := loadRuleSetCache(db, tag)
	if saved == nil || !bytes.Equal(saved.URLHash, ruleSetURLHash(url)) {
		return nil
	}
	return saved
}

func writeUvarint(w io.Writer, value uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	_, err := w.Write(buf[:n])
	return err
}

func readUvarint(r io.ByteReader) (uint64, error) {
	return binary.ReadUvarint(r)
}

func (u *RuleSetUpdater) openCacheReadOnly() (*bbolt.DB, error) {
	if _, err := os.Stat(u.cachePath); err != nil {
		return nil, err
	}
	return bbolt.Open(u.cachePath, 0600, &bbolt.Options{Timeout: time.Second, ReadOnly: true})
}

func (u *RuleSetUpdater) openCacheReadWrite() (*bbolt.DB, error) {
	if err := os.MkdirAll(filepath.Dir(u.cachePath), 0700); err != nil {
		return nil, err
	}
	return bbolt.Open(u.cachePath, 0600, &bbolt.Options{Timeout: 2 * time.Second})
}

func loadRuleSetCache(db *bbolt.DB, tag string) *savedRuleSetBinary {
	var saved savedRuleSetBinary
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(singBoxRuleSetBucket))
		if bucket == nil {
			return os.ErrNotExist
		}
		data := bucket.Get([]byte(tag))
		if len(data) == 0 {
			return os.ErrNotExist
		}
		return saved.UnmarshalBinary(data)
	})
	if err != nil {
		return nil
	}
	return &saved
}

func (u *RuleSetUpdater) saveRemoteCache(tag string, saved *savedRuleSetBinary) error {
	db, err := u.openCacheReadWrite()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuleSetCacheDisabled, err)
	}
	defer func() { _ = db.Close() }()
	payload, err := saved.MarshalBinary()
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(singBoxRuleSetBucket))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(tag), payload)
	})
}

func (u *RuleSetUpdater) touchRemoteCache(tag string, updated time.Time) error {
	db, err := u.openCacheReadWrite()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRuleSetCacheDisabled, err)
	}
	defer func() { _ = db.Close() }()
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(singBoxRuleSetBucket))
		if bucket == nil {
			return nil
		}
		data := bucket.Get([]byte(tag))
		if len(data) == 0 {
			return nil
		}
		var saved savedRuleSetBinary
		if err := saved.UnmarshalBinary(data); err != nil {
			return err
		}
		saved.LastUpdated = updated
		payload, err := saved.MarshalBinary()
		if err != nil {
			return err
		}
		return bucket.Put([]byte(tag), payload)
	})
}
