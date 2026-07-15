package api

import (
	"encoding/json"
	"errors"
	"os"
)

func readSyncConfig(configPath string) (map[string]any, []byte, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, err
	}
	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, nil, err
	}
	return config, data, nil
}

type managedGroupStore interface {
	SetURLTestManagedGroups([]string) error
}

type syncCommit struct {
	path     string
	previous []byte
	groups   managedGroupStore
}

func (c syncCommit) write(config map[string]any, groupTags []string) error {
	written, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWriteFile(c.path, written); err != nil {
		return err
	}
	if err := c.groups.SetURLTestManagedGroups(groupTags); err != nil {
		return errors.Join(err, atomicWriteFile(c.path, c.previous))
	}
	return nil
}
