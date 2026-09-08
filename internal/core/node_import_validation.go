package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	shadowsocks "github.com/sagernet/sing-shadowsocks2"
	"github.com/sagernet/sing-shadowsocks2/cipher"
	vmess "github.com/sagernet/sing-vmess"
	"github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/service"

	"github.com/xuthus5/boxd/internal/model"
)

// ValidateImportedNode 在保存前校验出站选项及可离线验证的协议字段，不创建内核或网络连接。
func ValidateImportedNode(node model.Outbound) error {
	body, err := importedNodeJSON(node)
	if err != nil {
		return errors.New("config: expected JSON outbound options object")
	}
	var outbound option.Outbound
	ctx := service.ContextWith[option.OutboundOptionsRegistry](context.Background(), include.OutboundRegistry())
	if err := outbound.UnmarshalJSONContext(ctx, body); err != nil {
		return errors.New("config: invalid outbound options")
	}
	switch options := outbound.Options.(type) {
	case *option.VLESSOutboundOptions:
		return validateImportedVLESS(options)
	case *option.VMessOutboundOptions:
		return validateImportedVMess(options)
	case *option.TUICOutboundOptions:
		return validateImportedTUIC(options)
	case *option.ShadowsocksOutboundOptions:
		return validateImportedShadowsocks(options)
	default:
		return nil
	}
}

func importedNodeJSON(node model.Outbound) ([]byte, error) {
	config := map[string]any{
		"type": node.Type, "tag": node.Tag, "server": node.Server, "server_port": node.Port,
	}
	if node.Raw != nil {
		body, err := json.Marshal(node.Raw)
		if err != nil {
			return nil, err
		}
		raw := map[string]any{}
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, err
		}
		// 与托管出站同步保持一致：原始选项覆盖表单字段。
		for key, value := range raw {
			config[key] = value
		}
	}
	return json.Marshal(config)
}

func validateImportedVLESS(options *option.VLESSOutboundOptions) error {
	// sing-vmess 会将非 UUID（含空串）映射为 UUID，复用构造器以保留兼容行为。
	if _, err := vless.NewClient(options.UUID, options.Flow, nil); err != nil {
		return errors.New("config.flow: unsupported VLESS flow")
	}
	return nil
}

func validateImportedVMess(options *option.VMessOutboundOptions) error {
	security := options.Security
	if security == "" {
		security = "auto"
	}
	if _, err := vmess.NewClient(options.UUID, security, options.AlterId); err != nil {
		return errors.New("config.security: unsupported VMess security type")
	}
	return nil
}

func validateImportedTUIC(options *option.TUICOutboundOptions) error {
	if _, err := uuid.FromString(options.UUID); err != nil {
		// UUID 也是认证信息，不回显解析器错误中包含的原值。
		return errors.New("config.uuid: invalid TUIC UUID")
	}
	return nil
}

func validateImportedShadowsocks(options *option.ShadowsocksOutboundOptions) error {
	_, err := shadowsocks.CreateMethod(context.Background(), options.Method, shadowsocks.MethodOptions{
		Password: options.Password,
	})
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), "unknown method:") {
		return errors.New("config.method: unsupported Shadowsocks method")
	}
	if errors.Is(err, cipher.ErrMissingPassword) {
		return errors.New("config.password: password is required for the selected Shadowsocks method")
	}
	return errors.New("config.password: password or key is invalid for the selected Shadowsocks method")
}
