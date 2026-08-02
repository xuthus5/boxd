package service

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/xuthus5/boxd/internal/core"
)

var outboundSyncMutex sync.Mutex

type outboundSyncSnapshot struct {
	configPath string
	config     []byte
	groups     []string
	settings   *core.SettingsManager
}

func captureOutboundSyncSnapshot(
	subManager *core.SubscriptionManager,
	configPath string,
) (outboundSyncSnapshot, error) {
	config, err := os.ReadFile(configPath)
	if err != nil {
		return outboundSyncSnapshot{}, err
	}
	settings := core.NewSettingsManager(subManager.DB())
	groups, err := settings.URLTestManagedGroups()
	if err != nil {
		return outboundSyncSnapshot{}, err
	}
	return outboundSyncSnapshot{
		configPath: configPath,
		config:     config,
		groups:     append([]string(nil), groups...),
		settings:   settings,
	}, nil
}

func (s outboundSyncSnapshot) restore() error {
	configErr := atomicWriteFile(s.configPath, s.config)
	if configErr != nil {
		configErr = fmt.Errorf("restoring previous outbound configuration: %w", configErr)
	}
	groupsErr := s.settings.SetURLTestManagedGroups(s.groups)
	if groupsErr != nil {
		groupsErr = fmt.Errorf("restoring previous managed groups: %w", groupsErr)
	}
	return errors.Join(configErr, groupsErr)
}

func syncOutboundsAndRestart(
	nodeManager *core.NodeManager,
	subManager *core.SubscriptionManager,
	configPath string,
	instance restartable,
) error {
	outboundSyncMutex.Lock()
	defer outboundSyncMutex.Unlock()

	snapshot, err := captureOutboundSyncSnapshot(subManager, configPath)
	if err != nil {
		return err
	}
	if err := SyncOutboundsToConfig(nodeManager, subManager, configPath); err != nil {
		return errors.Join(err, snapshot.restore())
	}
	changed, err := outboundConfigChanged(snapshot, configPath)
	if err != nil {
		return errors.Join(err, snapshot.restore())
	}
	if !changed || instance == nil {
		return nil
	}
	restartErr := instance.Restart()
	if restartErr == nil {
		return nil
	}
	slog.Error("auto-restart after outbound sync failed", "err", restartErr)
	rollbackErr := snapshot.restore()
	rollbackRestartErr := instance.Restart()
	if rollbackErr == nil && rollbackRestartErr == nil {
		return fmt.Errorf("restart failed after outbound sync; previous configuration restored: %w", restartErr)
	}
	result := []error{fmt.Errorf("restart failed after outbound sync: %w", restartErr)}
	if rollbackErr != nil {
		result = append(result, rollbackErr)
	}
	if rollbackRestartErr != nil {
		result = append(result, fmt.Errorf("restart failed after restoring previous configuration: %w", rollbackRestartErr))
	}
	return errors.Join(result...)
}

func outboundConfigChanged(snapshot outboundSyncSnapshot, configPath string) (bool, error) {
	current, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf("reading synchronized outbound configuration: %w", err)
	}
	return !bytes.Equal(snapshot.config, current), nil
}
