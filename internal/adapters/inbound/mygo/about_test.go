package mygo

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/internal/updatecheck"
)

func TestAboutPageShowsGitHubRepositoryAndOpensLinks(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageAbout
	tester := ui.NewTester(controller.View, 1320, 1200)

	for _, want := range []string{"GitHub 仓库", "xuthus5/boxd", "github.com/xuthus5/boxd", "Apache-2.0", "打开仓库", "问题反馈", "检查更新"} {
		if !tester.HasText(want) {
			t.Fatalf("about page missing %q; texts=%q", want, tester.Texts())
		}
	}
	if err := tester.Click("打开仓库"); err != nil {
		t.Fatalf("click repository: %v", err)
	}
	if err := tester.Click("问题反馈"); err != nil {
		t.Fatalf("click issues: %v", err)
	}
	opened := tester.OpenedURLs()
	if len(opened) != 2 || opened[0] != "https://github.com/xuthus5/boxd" || opened[1] != "https://github.com/xuthus5/boxd/issues" {
		t.Fatalf("opened URLs = %v", opened)
	}
}

func TestAboutUpdateAvailableRendersReleaseAction(t *testing.T) {
	controller := newViewTestController(ControllerOptions{UpdateChecker: staticUpdateChecker{}})
	controller.page = pageAbout
	controller.completeUpdate(updatecheck.Release{
		Tag:         "v0.3.0",
		Name:        "BOXD 0.3.0",
		URL:         "https://github.com/xuthus5/boxd/releases/tag/v0.3.0",
		PublishedAt: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
	}, nil)
	tester := ui.NewTester(controller.View, 1320, 1200)

	if controller.update.state != updateAvailable {
		t.Fatalf("update state = %v", controller.update.state)
	}
	if !tester.HasText("发现新版本 v0.3.0") || !tester.HasText("查看版本") {
		t.Fatalf("available update not rendered; texts=%q", tester.Texts())
	}
	if err := tester.Click("查看版本"); err != nil {
		t.Fatalf("click release: %v", err)
	}
	if opened := tester.OpenedURLs(); len(opened) != 1 || opened[0] != "https://github.com/xuthus5/boxd/releases/tag/v0.3.0" {
		t.Fatalf("opened URLs = %v", opened)
	}
}

func TestAboutNoStableReleaseIsAWarningNotFailure(t *testing.T) {
	controller := newViewTestController(ControllerOptions{UpdateChecker: staticUpdateChecker{}})
	controller.page = pageAbout
	controller.completeUpdate(updatecheck.Release{}, updatecheck.ErrNoRelease)
	tester := ui.NewTester(controller.View, 1320, 1200)

	if controller.update.state != updateNoRelease {
		t.Fatalf("update state = %v", controller.update.state)
	}
	if !tester.HasText("暂无正式版本") || tester.HasText("发现新版本") {
		t.Fatalf("no-release state not rendered correctly; texts=%q", tester.Texts())
	}
	if err := tester.Click("发布记录"); err != nil {
		t.Fatalf("click release history: %v", err)
	}
	if opened := tester.OpenedURLs(); len(opened) != 1 || opened[0] != "https://github.com/xuthus5/boxd/releases" {
		t.Fatalf("opened URLs = %v", opened)
	}
}

func TestAboutCheckUpdateStartsBackgroundRequest(t *testing.T) {
	checker := &blockingUpdateChecker{
		started: make(chan struct{}),
		finish:  make(chan struct{}),
		release: updatecheck.Release{Tag: "v0.4.0", URL: "https://github.com/xuthus5/boxd/releases/tag/v0.4.0"},
	}
	controller := newViewTestController(ControllerOptions{UpdateChecker: checker})
	controller.page = pageAbout
	tester := ui.NewTester(controller.View, 1320, 1200)

	if err := tester.Click("检查更新"); err != nil {
		t.Fatalf("click check updates: %v", err)
	}
	select {
	case <-checker.started:
	case <-time.After(2 * time.Second):
		t.Fatal("update checker was not invoked")
	}
	if controller.update.state != updateChecking {
		t.Fatalf("update state = %v, want checking", controller.update.state)
	}
	tester.Frame()
	if !tester.HasText("正在检查更新") {
		t.Fatalf("checking state not rendered; texts=%q", tester.Texts())
	}
	close(checker.finish)
}

type staticUpdateChecker struct{}

func (staticUpdateChecker) LatestRelease(context.Context) (updatecheck.Release, error) {
	return updatecheck.Release{}, updatecheck.ErrNoRelease
}

type blockingUpdateChecker struct {
	started chan struct{}
	finish  chan struct{}
	release updatecheck.Release
}

func (c *blockingUpdateChecker) LatestRelease(ctx context.Context) (updatecheck.Release, error) {
	close(c.started)
	select {
	case <-c.finish:
		return c.release, nil
	case <-ctx.Done():
		return updatecheck.Release{}, ctx.Err()
	}
}
