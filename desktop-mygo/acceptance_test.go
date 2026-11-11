package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type acceptancePage struct {
	name string
	path string
	text string
}

func TestAcceptanceScreenshots(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	fake, ok := app.backend.(*fakeBackend)
	if !ok {
		t.Fatal("acceptance backend is not a fake backend")
	}
	fake.configText = "{\n  \"log\": {\"level\": \"info\"},\n  \"inbounds\": []\n}\n"
	app.loadConfigSync()
	tt := ui.NewTester(app.view, 1440, 900)
	runID := acceptanceRunID()
	dir := filepath.Join(acceptanceRoot(), runID, "screenshots")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	pages := []acceptancePage{
		{name: "dashboard-light", path: "/dashboard", text: "Kernel controls"},
		{name: "nodes", path: "/nodes", text: "Node editing"},
		{name: "subscriptions", path: "/subscriptions", text: "Healthy"},
		{name: "connections", path: "/connections", text: "Active"},
		{name: "logs", path: "/logs", text: "Recent activity"},
		{name: "config", path: "/config", text: "Raw configuration"},
		{name: "settings", path: "/settings", text: "Appearance"},
	}
	var light *imageRGBA
	for _, page := range pages {
		app.router.Push(page.path)
		settleTransitions(tt)
		if !tt.HasText(page.text) {
			t.Fatalf("page %s is missing %q", page.path, page.text)
		}
		img := tt.Image()
		assertImage(t, img, 1440, 900)
		if page.name == "dashboard-light" {
			light = &imageRGBA{img: img}
		}
		writeScreenshot(t, filepath.Join(dir, page.name+".png"), img)
	}

	app.model.prefs.Theme = themeDark
	app.router.Replace("/dashboard")
	settleTransitions(tt)
	if !tt.HasText("Kernel controls") {
		t.Fatal("dark dashboard missing heading")
	}
	dark := tt.Image()
	assertImage(t, dark, 1440, 900)
	if light != nil && imageSignature(light.img) == imageSignature(dark) {
		t.Fatal("light and dark dashboard screenshots are identical")
	}
	writeScreenshot(t, filepath.Join(dir, "dashboard-dark.png"), dark)
	t.Logf("acceptance screenshots: %s", dir)
}

func TestAcceptanceEmptyStates(t *testing.T) {
	app := newTestApplication(newAppModel())
	app.model.loaded = true
	tt := ui.NewTester(app.view, 1440, 900)
	for _, page := range []acceptancePage{
		{name: "nodes-empty", path: "/nodes", text: "No nodes imported yet"},
		{name: "subscriptions-empty", path: "/subscriptions", text: "No subscriptions"},
		{name: "connections-empty", path: "/connections", text: "No active connections"},
		{name: "logs-empty", path: "/logs", text: "No matching log entries"},
	} {
		app.router.Replace(page.path)
		settleTransitions(tt)
		if !tt.HasText(page.text) {
			t.Fatalf("empty page %s is missing %q", page.path, page.text)
		}
		img := tt.Image()
		assertImage(t, img, 1440, 900)
		dir := filepath.Join(acceptanceRoot(), acceptanceRunID(), "screenshots")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeScreenshot(t, filepath.Join(dir, page.name+".png"), img)
	}
}

func settleTransitions(tt *ui.Tester) {
	tt.Frame()
	if tt.FramePending() {
		tt.Advance(2 * time.Second)
	}
}

type imageRGBA struct{ img *image.RGBA }

func acceptanceRoot() string {
	if root := os.Getenv("MYGO_ACCEPTANCE_DIR"); root != "" {
		return root
	}
	return filepath.Join("artifacts", "acceptance")
}

func acceptanceRunID() string {
	if id := os.Getenv("MYGO_ACCEPTANCE_RUN_ID"); id != "" {
		return id
	}
	return time.Now().Format("20060102-150405")
}

func assertImage(t *testing.T, img *image.RGBA, width, height int) {
	t.Helper()
	if img == nil || img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Fatalf("image bounds = %v, want %dx%d", img.Bounds(), width, height)
	}
	seen := make(map[color.RGBA]struct{})
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 16 {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 16 {
			seen[color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)] = struct{}{}
		}
	}
	if len(seen) < 3 {
		t.Fatalf("image is effectively blank: %d colors", len(seen))
	}
}

func imageSignature(img *image.RGBA) string {
	return fmt.Sprintf("%v", color.RGBAModel.Convert(img.At(img.Bounds().Min.X+10, img.Bounds().Min.Y+10)))
}

func writeScreenshot(t *testing.T, path string, img *image.RGBA) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}
