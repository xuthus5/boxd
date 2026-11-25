package httpapi

import (
	"net/http"

	"github.com/xuthus5/boxd/control"
)

type dnsPatch struct {
	Strategy *string                  `json:"strategy,omitempty"`
	Cache    *bool                    `json:"cache,omitempty"`
	Domestic *control.DNSServerConfig `json:"domestic,omitempty"`
	Foreign  *control.DNSServerConfig `json:"foreign,omitempty"`
	Default  *control.DNSServerConfig `json:"default,omitempty"`
}

type settingsUpdateRequest struct {
	Settings map[string]string `json:"settings"`
}

func (s *Server) registerDNSRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dns", s.handleDNSGet)
	mux.HandleFunc("PUT /api/v1/dns", s.handleDNSPut)
	mux.HandleFunc("PATCH /api/v1/dns", s.handleDNSPatch)
	mux.HandleFunc("POST /api/v1/dns/flush", s.handleDNSFlush)
	mux.HandleFunc("POST /api/v1/dns/test", s.handleDNSTest)
}

func (s *Server) registerSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", s.handleSettingsList)
	mux.HandleFunc("PUT /api/v1/settings", s.handleSettingsUpdate)
	mux.HandleFunc("GET /api/v1/settings/{key}", s.handleSettingGet)
	mux.HandleFunc("PUT /api/v1/settings/{key}", s.handleSettingSet)
}

func (s *Server) handleDNSGet(w http.ResponseWriter, r *http.Request) {
	profile, err := s.app.DNS().Get(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s.dnsView(r, profile))
}

func (s *Server) handleDNSPut(w http.ResponseWriter, r *http.Request) {
	var profile control.DNSProfile
	if err := decodeJSON(w, r, &profile); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.DNS().Update(r.Context(), profile)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "dns", "DNS profile updated", nil)
	writeJSON(w, http.StatusOK, s.dnsView(r, saved))
}

func (s *Server) handleDNSPatch(w http.ResponseWriter, r *http.Request) {
	var patch dnsPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.DNS().Patch(r.Context(), control.DNSProfilePatch{
		Strategy: patch.Strategy, Cache: patch.Cache, Domestic: patch.Domestic, Foreign: patch.Foreign, Default: patch.Default,
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "dns", "DNS profile updated", nil)
	writeJSON(w, http.StatusOK, s.dnsView(r, saved))
}

func (s *Server) handleDNSFlush(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DNS().Flush(r.Context()); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "dns", "DNS cache flushed", nil)
	writeJSON(w, http.StatusOK, map[string]bool{"flushed": true})
}

func (s *Server) handleDNSTest(w http.ResponseWriter, r *http.Request) {
	profile, err := s.app.DNS().Get(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	if _, err := s.app.DNS().Test(r.Context()); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "dns", "DNS latency probe completed", nil)
	writeJSON(w, http.StatusOK, s.dnsView(r, profile))
}

func (s *Server) dnsView(r *http.Request, profile control.DNSProfile) dnsProfileView {
	probes, err := s.app.DNS().ProbeResults(r.Context())
	if err != nil {
		return makeDNSView(profile)
	}
	return makeDNSViewWithProbes(profile, probes)
}

func (s *Server) handleSettingsList(w http.ResponseWriter, r *http.Request) {
	settings, err := s.app.Settings().List(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeSettingsMap(settings))
}

func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var request settingsUpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	if len(request.Settings) == 0 || len(request.Settings) > 100 {
		writeError(w, r, newAPIError(http.StatusBadRequest, "invalid_settings", "settings.update", "settings must contain between 1 and 100 values", "check the request body"))
		return
	}
	for key, value := range request.Settings {
		if _, err := s.app.Settings().Set(r.Context(), key, value); err != nil {
			s.writeServiceError(w, r, err)
			return
		}
	}
	s.logs.add("info", "settings", "settings updated", map[string]any{"keys": len(request.Settings)})
	settings, err := s.app.Settings().List(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeSettingsMap(settings))
}

func (s *Server) handleSettingGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	setting, found, err := s.app.Settings().Get(r.Context(), key)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, newAPIError(http.StatusNotFound, "kernel_not_found", "settings.get", "setting was not found", "use a known setting key"))
		return
	}
	writeJSON(w, http.StatusOK, setting)
}

func (s *Server) handleSettingSet(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Value string `json:"value"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	setting, err := s.app.Settings().Set(r.Context(), r.PathValue("key"), request.Value)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, setting)
}
