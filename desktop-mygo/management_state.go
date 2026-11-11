package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

type nodeFormState struct {
	open       bool
	edit       bool
	editingTag string
	link       string
	tag        string
	typ        string
	server     string
	port       float64
	config     string
	err        string
}

type subscriptionFormState struct {
	open         bool
	edit         bool
	id           string
	name         string
	url          string
	interval     float64
	urlTestMode  string
	testURL      string
	testInterval string
	tolerance    float64
	err          string
}

const (
	urlTestModeInherit  = "Inherit global"
	urlTestModeEnabled  = "Enable URL-test"
	urlTestModeDisabled = "Disable URL-test"
)

type selectorFormState struct {
	open           bool
	subscriptionID string
	name           string
	selected       string
	options        []string
}

type deleteFormState struct {
	open    bool
	kind    string
	id      string
	ids     []string
	title   string
	message string
	confirm string
}

func (s *nodeFormState) reset() {
	*s = nodeFormState{port: 443, config: "{}", open: true}
}

func (s *nodeFormState) beginEdit(node nodeView) {
	*s = nodeFormState{
		open: true, edit: true, editingTag: node.Tag, tag: node.Tag, typ: node.Type,
		server: node.Server, port: float64(node.Port), config: formatJSONValue(node.Config),
	}
}

func (s *nodeFormState) input() (service.NodeInput, error) {
	tag := strings.TrimSpace(s.tag)
	typ := strings.TrimSpace(s.typ)
	server := strings.TrimSpace(s.server)
	port := int(s.port)
	if tag == "" || typ == "" || server == "" || port < 1 || port > 65535 {
		return service.NodeInput{}, fmt.Errorf("tag, type, server and a valid port are required")
	}
	var config any
	decoder := json.NewDecoder(strings.NewReader(s.config))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return service.NodeInput{}, fmt.Errorf("invalid node config JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return service.NodeInput{}, fmt.Errorf("invalid node config JSON: trailing data")
	}
	return service.NodeInput{Tag: tag, Type: typ, Server: server, Port: port, Config: config}, nil
}

func (s *subscriptionFormState) reset() {
	*s = subscriptionFormState{
		open: true, interval: 60, urlTestMode: urlTestModeInherit,
		testInterval: "3m", tolerance: 50,
	}
}

func (s *subscriptionFormState) beginEdit(sub model.Subscription) {
	*s = subscriptionFormState{
		open: true, edit: true, id: sub.ID, name: sub.Name, url: sub.URL,
		interval: float64(sub.IntervalMin), urlTestMode: urlTestModeInherit,
		testInterval: "3m", tolerance: 50,
	}
	if sub.URLTest != nil {
		if sub.URLTest.Enabled == nil {
			s.urlTestMode = urlTestModeInherit
		} else if *sub.URLTest.Enabled {
			s.urlTestMode = urlTestModeEnabled
		} else {
			s.urlTestMode = urlTestModeDisabled
		}
		if sub.URLTest.URL != nil {
			s.testURL = *sub.URLTest.URL
		}
		if sub.URLTest.Interval != nil {
			s.testInterval = *sub.URLTest.Interval
		}
		if sub.URLTest.Tolerance != nil {
			s.tolerance = float64(*sub.URLTest.Tolerance)
		}
	}
}

func (s *subscriptionFormState) input() (service.SubscriptionInput, error) {
	if strings.TrimSpace(s.name) == "" || strings.TrimSpace(s.url) == "" {
		return service.SubscriptionInput{}, fmt.Errorf("name and URL are required")
	}
	input := service.SubscriptionInput{
		Name: strings.TrimSpace(s.name), URL: strings.TrimSpace(s.url), IntervalMin: int(s.interval),
	}
	if s.urlTestMode == urlTestModeInherit {
		return input, nil
	}
	enabled := s.urlTestMode == urlTestModeEnabled
	input.URLTest = &model.URLTestOverrides{Enabled: &enabled}
	if enabled {
		interval := strings.TrimSpace(s.testInterval)
		tolerance := uint16(s.tolerance)
		input.URLTest.Interval = &interval
		input.URLTest.Tolerance = &tolerance
		if url := strings.TrimSpace(s.testURL); url != "" {
			input.URLTest.URL = &url
		}
	}
	return input, nil
}

func formatJSONValue(value any) string {
	if value == nil {
		return "{}"
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(body)
}
