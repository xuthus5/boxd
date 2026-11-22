package kernel

import "context"

// ConfigDiagnosticsFeature provides human-readable configuration diagnostics.
type ConfigDiagnosticsFeature interface {
	DiagnoseConfig(ctx context.Context, doc ConfigDocument) ([]Diagnostic, error)
}

// DefaultsFeature installs engine-specific defaults into an opaque document.
type DefaultsFeature interface {
	InstallDefaults(ctx context.Context, doc ConfigDocument, section string) (ConfigDocument, any, error)
}

// NodeCodecFeature converts share links without leaking engine types.
type NodeCodecFeature interface {
	DecodeNodeLink(ctx context.Context, link string) (NodeSpec, error)
	EncodeNodeLink(ctx context.Context, spec NodeSpec) (string, error)
}
