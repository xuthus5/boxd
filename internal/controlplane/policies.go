package controlplane

import (
	"context"

	"github.com/xuthus5/boxd/control"
)

// PolicyService provides a read-only aggregate over routing and DNS modules.
type PolicyService struct {
	routes control.RouteAPI
	dns    control.DNSAPI
}

var _ control.PolicyAPI = (*PolicyService)(nil)

func NewPolicyService(routes control.RouteAPI, dns control.DNSAPI) *PolicyService {
	return &PolicyService{routes: routes, dns: dns}
}

func (s *PolicyService) Get(ctx context.Context) (control.PolicySnapshot, error) {
	routes, err := s.routes.Get(ctx)
	if err != nil {
		return control.PolicySnapshot{}, err
	}
	dns, err := s.dns.Get(ctx)
	if err != nil {
		return control.PolicySnapshot{}, err
	}
	return control.PolicySnapshot{Routes: routes, DNS: dns}, nil
}

func (s *PolicyService) Routes() control.RouteAPI { return s.routes }
func (s *PolicyService) DNS() control.DNSAPI      { return s.dns }
