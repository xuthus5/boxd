package nodelink

import (
	"context"
	"encoding/json"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

var _ kernel.NodeCodecFeature = (*Codec)(nil)

// DecodeNodeLink implements the kernel optional feature. Protocol-specific
// options are preserved as JSON metadata so another engine adapter can consume
// the same link without importing this package.
func (c *Codec) DecodeNodeLink(ctx context.Context, link string) (kernel.NodeSpec, error) {
	node, err := c.Decode(ctx, link)
	if err != nil {
		return kernel.NodeSpec{}, err
	}
	metadata := map[string]string{"name": node.Name}
	if len(node.Options) > 0 {
		options, err := json.Marshal(node.Options)
		if err != nil {
			return kernel.NodeSpec{}, linkError("nodes.decode_link", "encode node options: "+err.Error())
		}
		metadata["options"] = string(options)
	}
	return kernel.NodeSpec{
		ID:       node.ID,
		Tag:      node.Name,
		Protocol: node.Protocol,
		Server:   node.Server,
		Port:     node.Port,
		Metadata: metadata,
	}, nil
}

// EncodeNodeLink implements the kernel optional feature.
func (c *Codec) EncodeNodeLink(ctx context.Context, spec kernel.NodeSpec) (string, error) {
	node := control.Node{
		ID:       spec.ID,
		Name:     spec.Tag,
		Protocol: spec.Protocol,
		Server:   spec.Server,
		Port:     spec.Port,
	}
	if name := spec.Metadata["name"]; name != "" {
		node.Name = name
	}
	if raw := spec.Metadata["options"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &node.Options); err != nil {
			return "", linkError("nodes.encode_link", "decode node options: "+err.Error())
		}
	}
	return c.Encode(ctx, node)
}
