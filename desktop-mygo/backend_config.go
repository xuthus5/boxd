package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

const (
	sectionInbounds            = "inbounds"
	sectionOutbounds           = "outbounds"
	sectionRoute               = "route"
	sectionDNS                 = "dns"
	sectionEndpoints           = "endpoints"
	sectionServices            = "services"
	sectionCertificate         = "certificate"
	sectionLog                 = "log"
	sectionNTP                 = "ntp"
	sectionExperimental        = "experimental"
	sectionNetworkNamespaces   = "network_namespaces"
	sectionHTTPClients         = "http_clients"
	sectionCertificateProvider = "certificate_providers"
)

var configSectionKeys = map[string]struct{}{
	sectionInbounds: {}, sectionOutbounds: {}, sectionRoute: {}, sectionDNS: {},
	sectionEndpoints: {}, sectionServices: {}, sectionCertificate: {}, sectionLog: {},
	sectionNTP: {}, sectionExperimental: {}, sectionNetworkNamespaces: {},
	sectionHTTPClients: {}, sectionCertificateProvider: {},
}

var configSectionArrays = map[string]struct{}{
	sectionInbounds: {}, sectionOutbounds: {}, sectionEndpoints: {}, sectionServices: {},
	sectionNetworkNamespaces: {}, sectionHTTPClients: {}, sectionCertificateProvider: {},
}

func validateConfigSection(section string) error {
	if _, ok := configSectionKeys[section]; !ok {
		return fmt.Errorf("unsupported config section %q", section)
	}
	return nil
}

func emptyConfigSection(section string) string {
	if _, ok := configSectionArrays[section]; ok {
		return "[]\n"
	}
	return "{}\n"
}

func (b *runtimeBackend) LoadConfigSection(ctx context.Context, section string) (string, error) {
	if err := validateConfigSection(section); err != nil {
		return "", err
	}
	svc, err := b.serviceSet()
	if err != nil {
		return "", err
	}
	config, err := svc.Config().GetConfig(ctx)
	if err != nil {
		return "", err
	}
	root, ok := config.(map[string]any)
	if !ok {
		return "", errors.New("configuration root is not an object")
	}
	value, ok := root[section]
	if !ok {
		return emptyConfigSection(section), nil
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode config section: %w", err)
	}
	return string(body) + "\n", nil
}

func (b *runtimeBackend) ApplyConfigSection(ctx context.Context, section, body string) error {
	full, err := b.mergedConfigSection(ctx, section, body)
	if err != nil {
		return err
	}
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Config().ApplyConfig(ctx, full, "desktop-mygo:"+section)
	return err
}

func (b *runtimeBackend) ValidateConfigSection(ctx context.Context, section, body string) error {
	full, err := b.mergedConfigSection(ctx, section, body)
	if err != nil {
		return err
	}
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Config().ValidateConfig(ctx, full, "desktop-mygo:"+section)
}

func (b *runtimeBackend) mergedConfigSection(ctx context.Context, section, body string) ([]byte, error) {
	if err := validateConfigSection(section); err != nil {
		return nil, err
	}
	value, err := decodeJSONValue(body)
	if err != nil {
		return nil, err
	}
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	config, err := svc.Config().GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	root, ok := config.(map[string]any)
	if !ok {
		return nil, errors.New("configuration root is not an object")
	}
	root[section] = value
	full, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode configuration: %w", err)
	}
	return full, nil
}

func decodeJSONValue(body string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid JSON: trailing data")
	}
	return value, nil
}

func (b *runtimeBackend) InstallDefaultConfigSection(ctx context.Context, section string) error {
	if err := validateConfigSection(section); err != nil {
		return err
	}
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	switch section {
	case sectionInbounds:
		_, err = svc.Config().InstallDefaultInbounds(ctx)
	case sectionOutbounds:
		_, err = svc.Config().InstallDefaultOutbounds(ctx)
	case sectionRoute:
		_, err = svc.Config().InstallDefaultRouteRules(ctx)
	case sectionDNS:
		_, err = svc.Config().InstallDefaultDNS(ctx)
	case sectionExperimental:
		_, err = svc.Config().InstallDefaultExperimental(ctx)
	default:
		return fmt.Errorf("no default installer for config section %q", section)
	}
	return err
}

func (b *runtimeBackend) ConfigDiagnostics(ctx context.Context) (model.ConfigDiagnostics, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return model.ConfigDiagnostics{}, err
	}
	return svc.Config().Diagnostics(), nil
}

func (b *runtimeBackend) ConfigApplyHistory(ctx context.Context) ([]model.ConfigApplyEvent, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	return svc.Config().ApplyHistory()
}

func (b *runtimeBackend) ConfigSnapshot(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("snapshot id is required")
	}
	svc, err := b.serviceSet()
	if err != nil {
		return "", err
	}
	body, err := svc.Config().ConfigSnapshot(id)
	if err != nil {
		return "", err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, body, "", "  "); err != nil {
		return string(body), nil
	}
	return formatted.String() + "\n", nil
}

func (b *runtimeBackend) RestoreConfigSnapshot(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("snapshot id is required")
	}
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Config().RestoreConfigSnapshot(ctx, id)
	return err
}

func (b *runtimeBackend) RuleSetStatus(ctx context.Context) ([]model.RuleSetStatusItem, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	return svc.RuleSets().Status(ctx)
}

func (b *runtimeBackend) UpdateRuleSets(ctx context.Context, req core.RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return model.RuleSetUpdateResponse{}, err
	}
	return svc.RuleSets().Update(ctx, req)
}

func (b *runtimeBackend) RuleSetAutoUpdate(ctx context.Context) (model.RuleSetAutoUpdate, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return model.RuleSetAutoUpdate{}, err
	}
	return svc.RuleSets().AutoUpdate()
}

func (b *runtimeBackend) SetRuleSetAutoUpdate(ctx context.Context, cfg model.RuleSetAutoUpdate) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.RuleSets().SetAutoUpdate(cfg)
}

func (b *runtimeBackend) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Settings().ChangePassword(ctx, currentPassword, newPassword)
	return err
}

func (b *runtimeBackend) CreateBackup(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("backup path is required")
	}
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Backup().CreateBackupArchiveTo(ctx, path)
}
