package httpapi

import "github.com/xuthus5/boxd/control"

// The web console consumes the transport-neutral control-plane observation
// model directly. These aliases keep the presentation helpers readable without
// duplicating aggregation logic in the HTTP adapter.
type observedTraffic = control.ObservedTraffic

type observationSnapshot = control.ObservationSnapshot
