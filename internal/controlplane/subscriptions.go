package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// SubscriptionService coordinates conditional downloads and atomic node
// replacement through the profile state transaction.
type SubscriptionService struct {
	profile    *ProfileService
	downloader control.Downloader
	parser     control.SubscriptionParser
	settings   control.SettingsAPI
	now        func() time.Time
}

var _ control.SubscriptionAPI = (*SubscriptionService)(nil)

func NewSubscriptionService(profile *ProfileService, downloader control.Downloader, parser control.SubscriptionParser, settings control.SettingsAPI) *SubscriptionService {
	return &SubscriptionService{profile: profile, downloader: downloader, parser: parser, settings: settings, now: time.Now}
}

func (s *SubscriptionService) List(ctx context.Context) ([]control.Subscription, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return append([]control.Subscription(nil), state.Subscriptions...), nil
}

func (s *SubscriptionService) Get(ctx context.Context, id string) (control.Subscription, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.Subscription{}, err
	}
	for _, subscription := range state.Subscriptions {
		if subscription.ID == id {
			return subscription, nil
		}
	}
	return control.Subscription{}, notFound("subscription", id)
}

func (s *SubscriptionService) Save(ctx context.Context, subscription control.Subscription) (control.Subscription, error) {
	subscription.ID = strings.TrimSpace(subscription.ID)
	if subscription.ID == "" {
		id, err := s.profile.nextID("sub")
		if err != nil {
			return control.Subscription{}, kernel.WrapError(kernel.ErrCodeInternal, "subscriptions.save", "generate subscription ID", err)
		}
		subscription.ID = id
	}
	state, err := s.profile.update(ctx, "subscriptions.save", func(state *control.DesiredState) error {
		for index := range state.Subscriptions {
			if state.Subscriptions[index].ID == subscription.ID {
				state.Subscriptions[index] = subscription
				return nil
			}
		}
		state.Subscriptions = append(state.Subscriptions, subscription)
		return nil
	})
	if err != nil {
		return control.Subscription{}, err
	}
	for _, saved := range state.Subscriptions {
		if saved.ID == subscription.ID {
			return saved, nil
		}
	}
	return control.Subscription{}, notFound("subscription", subscription.ID)
}

func (s *SubscriptionService) Delete(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, "subscriptions.delete", func(state *control.DesiredState) error {
		subscriptionIndex := -1
		for index := range state.Subscriptions {
			if state.Subscriptions[index].ID == id {
				subscriptionIndex = index
				break
			}
		}
		if subscriptionIndex < 0 {
			return notFound("subscription", id)
		}
		state.Subscriptions = append(state.Subscriptions[:subscriptionIndex], state.Subscriptions[subscriptionIndex+1:]...)
		nodes := state.Nodes[:0]
		for _, node := range state.Nodes {
			if node.SubscriptionID != id {
				nodes = append(nodes, node)
			}
		}
		state.Nodes = nodes
		return nil
	})
	return err
}

func (s *SubscriptionService) Refresh(ctx context.Context, id string) (control.SubscriptionRefreshResult, error) {
	if s.downloader == nil {
		return control.SubscriptionRefreshResult{}, capabilityUnavailable("subscriptions.refresh", "subscription downloader")
	}
	if s.parser == nil {
		return control.SubscriptionRefreshResult{}, capabilityUnavailable("subscriptions.refresh", "subscription parser")
	}
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.SubscriptionRefreshResult{}, err
	}
	var subscription control.Subscription
	found := false
	for _, candidate := range state.Subscriptions {
		if candidate.ID == id {
			subscription = candidate
			found = true
			break
		}
	}
	if !found {
		return control.SubscriptionRefreshResult{}, notFound("subscription", id)
	}
	if strings.TrimSpace(subscription.URL) == "" {
		return control.SubscriptionRefreshResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "subscriptions.refresh", "subscription URL is required")
	}

	result, err := s.downloader.Get(ctx, control.DownloadRequest{
		URL:          subscription.URL,
		ETag:         subscription.ETag,
		LastModified: subscription.LastModified,
		Timeout:      defaultDownloadTimeout * time.Second,
		MaxBytes:     defaultDownloadBytes,
	})
	if err != nil {
		return control.SubscriptionRefreshResult{}, wrapDownloadError("subscriptions.refresh", err)
	}

	refresh := control.SubscriptionRefreshResult{Subscription: subscription, NotModified: result.NotModified}
	var parsed []control.Node
	if !result.NotModified {
		parsed, err = s.parser.Parse(ctx, result)
		if err != nil {
			return control.SubscriptionRefreshResult{}, err
		}
		if len(parsed) == 0 {
			return control.SubscriptionRefreshResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "subscriptions.refresh", "subscription did not contain any proxy nodes")
		}
		parsed = normalizeSubscriptionNodes(id, parsed)
	}

	now := s.now().UTC()
	keepLocal := s.keepLocalNodes(ctx)
	_, err = s.profile.update(ctx, "subscriptions.refresh", func(state *control.DesiredState) error {
		subscriptionIndex := -1
		for index := range state.Subscriptions {
			if state.Subscriptions[index].ID == id {
				subscriptionIndex = index
				break
			}
		}
		if subscriptionIndex < 0 {
			return notFound("subscription", id)
		}
		updated := state.Subscriptions[subscriptionIndex]
		updated.LastRefreshedAt = now
		if result.ETag != "" {
			updated.ETag = result.ETag
		}
		if result.LastModified != "" {
			updated.LastModified = result.LastModified
		}
		state.Subscriptions[subscriptionIndex] = updated
		refresh.Subscription = updated
		if result.NotModified {
			return nil
		}

		previous := make(map[string]control.Node)
		remaining := make([]control.Node, 0, len(state.Nodes)+len(parsed))
		for _, node := range state.Nodes {
			if node.SubscriptionID == id {
				previous[node.ID] = node
				continue
			}
			remaining = append(remaining, node)
		}
		for _, node := range parsed {
			if _, exists := previous[node.ID]; exists {
				refresh.Updated++
			} else {
				refresh.Added++
			}
			delete(previous, node.ID)
		}
		if keepLocal {
			for _, node := range previous {
				remaining = append(remaining, node)
			}
			refresh.Removed = 0
		} else {
			refresh.Removed = len(previous)
		}
		state.Nodes = append(remaining, parsed...)
		return nil
	})
	if err != nil {
		return control.SubscriptionRefreshResult{}, err
	}
	return refresh, nil
}

func (s *SubscriptionService) keepLocalNodes(ctx context.Context) bool {
	if s.settings == nil {
		return true
	}
	setting, found, err := s.settings.Get(ctx, control.SettingSubscriptionConflictPolicy)
	if err != nil || !found {
		return true
	}
	return !strings.EqualFold(strings.TrimSpace(setting.Value), control.SubscriptionConflictFollowRemote)
}

func (s *SubscriptionService) RefreshAll(ctx context.Context) ([]control.SubscriptionRefreshResult, error) {
	subscriptions, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]control.SubscriptionRefreshResult, 0, len(subscriptions))
	var failures []error
	for _, subscription := range subscriptions {
		result, refreshErr := s.Refresh(ctx, subscription.ID)
		if refreshErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", subscription.ID, refreshErr))
			continue
		}
		results = append(results, result)
	}
	return results, errors.Join(failures...)
}

func normalizeSubscriptionNodes(subscriptionID string, nodes []control.Node) []control.Node {
	seen := make(map[string]struct{}, len(nodes))
	result := make([]control.Node, 0, len(nodes))
	for _, node := range nodes {
		node.SubscriptionID = subscriptionID
		node.Protocol = strings.ToLower(strings.TrimSpace(node.Protocol))
		node.Server = strings.TrimSpace(node.Server)
		node.Name = strings.TrimSpace(node.Name)
		node.ID = scopedStableID("subnode", subscriptionID, node.Protocol, node.Server, node.Port, node.Name)
		if _, exists := seen[node.ID]; exists {
			continue
		}
		seen[node.ID] = struct{}{}
		result = append(result, node)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func wrapDownloadError(op string, err error) error {
	var kernelError *kernel.Error
	if errors.As(err, &kernelError) {
		return err
	}
	return kernel.WrapError(kernel.ErrCodeInternal, op, "download remote content", err)
}
