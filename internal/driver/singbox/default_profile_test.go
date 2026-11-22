package singbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/singboxconfig"
	"github.com/xuthus5/boxd/kernel"
)

func TestGeneratedDefaultProfileStarts(t *testing.T) {
	state := control.DefaultDesiredState()
	state.Inbounds[0].Listen.Port = 18080
	state.Routes.Presets.ChinaDirect = false
	state.Routes.Presets.AdBlock = false
	document, err := singboxconfig.New().Compile(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseOptions(context.Background(), document, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	engine := newEngine(kernelOpenOptions(t.TempDir(), document), parsed)
	if err := engine.Start(context.Background()); err != nil {
		var messages []string
		for current := err; current != nil; current = errors.Unwrap(current) {
			messages = append(messages, current.Error())
		}
		t.Fatalf("start: %s\n%s", strings.Join(messages, " -> "), document.Body)
	}
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
}

func kernelOpenOptions(dir string, document kernel.ConfigDocument) kernel.OpenOptions {
	return kernel.OpenOptions{Name: "diag", WorkDir: dir, Config: document}
}
