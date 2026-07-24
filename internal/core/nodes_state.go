package core

import (
	"encoding/json"
	"strings"

	"go.etcd.io/bbolt"
)

type resultEntry struct {
	oldKey []byte
	newKey []byte
	value  []byte
}

func resultEntryKeys(tag string, key []byte) (string, bool) {
	keyString := string(key)
	if keyString == tag {
		return "", true
	}
	prefix := tag + "_"
	if strings.HasPrefix(keyString, prefix) {
		return keyString[len(tag):], true
	}
	return "", false
}

func collectResultEntries(bucket *bbolt.Bucket, oldTag, newTag string) ([]resultEntry, error) {
	if bucket == nil {
		return nil, nil
	}
	entries := make([]resultEntry, 0)
	err := bucket.ForEach(func(key, value []byte) error {
		suffix, ok := resultEntryKeys(oldTag, key)
		if !ok {
			return nil
		}
		entries = append(entries, resultEntry{
			oldKey: append([]byte(nil), key...),
			newKey: append([]byte(newTag), suffix...),
			value:  append([]byte(nil), value...),
		})
		return nil
	})
	return entries, err
}

func rewriteResultTag(data []byte, tag string) ([]byte, error) {
	var stored StoredResult
	if err := json.Unmarshal(data, &stored); err != nil || stored.Results == nil {
		return append([]byte(nil), data...), nil
	}
	for testType, result := range stored.Results {
		result.Tag = tag
		stored.Results[testType] = result
	}
	return json.Marshal(stored)
}

func moveResultEntries(bucket *bbolt.Bucket, oldTag, newTag string) error {
	entries, err := collectResultEntries(bucket, oldTag, newTag)
	if err != nil || bucket == nil {
		return err
	}
	for _, entry := range entries {
		if err := bucket.Delete(entry.oldKey); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		value, err := rewriteResultTag(entry.value, newTag)
		if err != nil {
			return err
		}
		if err := bucket.Put(entry.newKey, value); err != nil {
			return err
		}
	}
	return nil
}

func moveHistoryEntry(bucket *bbolt.Bucket, oldTag, newTag string) error {
	if bucket == nil {
		return nil
	}
	value := bucket.Get([]byte(oldTag))
	if value == nil {
		return nil
	}
	copyValue := append([]byte(nil), value...)
	if err := bucket.Delete([]byte(oldTag)); err != nil {
		return err
	}
	return bucket.Put([]byte(newTag), copyValue)
}

func deleteResultEntries(bucket *bbolt.Bucket, tag string) error {
	if bucket == nil {
		return nil
	}
	keys := make([][]byte, 0)
	err := bucket.ForEach(func(key, _ []byte) error {
		if _, ok := resultEntryKeys(tag, key); ok {
			keys = append(keys, append([]byte(nil), key...))
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := bucket.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
