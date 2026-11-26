package mygo

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTransientNoticesExpireWhileBusyNoticesStay(t *testing.T) {
	controller := NewController(nil, ControllerOptions{})
	controller.setNotice("启动内核完成", false)
	if controller.noticeUntil.IsZero() {
		t.Fatal("completed notice has no expiry")
	}
	controller.noticeUntil = time.Now().Add(-time.Second)
	controller.expireNotice()
	if controller.notice != "" {
		t.Fatalf("expired notice = %q, want cleared", controller.notice)
	}

	controller.busy = true
	controller.setNotice("启动内核中…", false)
	if !controller.noticeUntil.IsZero() {
		t.Fatal("busy notice should remain until the operation completes")
	}
	controller.expireNotice()
	if controller.notice == "" {
		t.Fatal("busy notice expired while the operation was still running")
	}
}

func TestStickyNoticeRequiresExplicitDismissal(t *testing.T) {
	controller := NewController(nil, ControllerOptions{})
	controller.setStickyNotice("控制面初始化失败", true)
	controller.expireNotice()
	if controller.notice == "" {
		t.Fatal("sticky notice expired without user action")
	}
	controller.clearNotice()
	if controller.notice != "" || controller.noticeErr || !controller.noticeUntil.IsZero() {
		t.Fatalf("notice state after clear = %q, %v, %v", controller.notice, controller.noticeErr, controller.noticeUntil)
	}
}

func TestExpiredNoticeRemovesBannerFromView(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.setNotice("停止内核完成", false)
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("停止内核完成") {
		t.Fatalf("notice banner missing; texts=%q", tester.Texts())
	}
	controller.noticeUntil = time.Now().Add(-time.Second)
	controller.expireNotice()
	tester.Frame()
	if tester.HasText("停止内核完成") {
		t.Fatalf("expired notice still rendered; texts=%q", tester.Texts())
	}
}
