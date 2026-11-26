package mygo

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/updatecheck"
)

type updateCheckState int

const (
	updateIdle updateCheckState = iota
	updateChecking
	updateCurrent
	updateAvailable
	updateNoRelease
	updateUnknown
	updateFailed
)

type updateStatus struct {
	state     updateCheckState
	release   updatecheck.Release
	checkedAt time.Time
	err       string
}

func (c *Controller) updateTitle() string {
	if c.options.UpdateChecker == nil {
		return "更新检查不可用"
	}
	switch c.update.state {
	case updateChecking:
		return "正在检查更新"
	case updateCurrent:
		return "已是最新版本"
	case updateAvailable:
		return "发现新版本 " + c.update.release.Tag
	case updateNoRelease:
		return "暂无正式版本"
	case updateUnknown:
		return "已获取最新版本"
	case updateFailed:
		return "检查失败"
	default:
		return "尚未检查更新"
	}
}

func (c *Controller) updateDetail() string {
	if c.options.UpdateChecker == nil {
		return "当前构建未配置更新检查服务"
	}
	switch c.update.state {
	case updateChecking:
		return "正在连接 GitHub Release…"
	case updateCurrent:
		return fmt.Sprintf("当前版本 %s · %s", c.version(), updateCheckedLabel(c.update.checkedAt))
	case updateAvailable:
		detail := "当前版本 " + c.version()
		if !c.update.release.PublishedAt.IsZero() {
			detail += " · 发布于 " + c.update.release.PublishedAt.Local().Format("2006-01-02")
		}
		return detail
	case updateNoRelease:
		return "GitHub 上还没有正式 Release，仍可打开发布记录查看"
	case updateUnknown:
		return "最新正式版本为 " + c.update.release.Tag + "；当前构建版本无法参与比较"
	case updateFailed:
		if detail := strings.TrimSpace(c.update.err); detail != "" {
			return detail
		}
		return "GitHub 请求失败，请检查网络连接后重试"
	default:
		return "检查 GitHub 上最新的正式版本"
	}
}

func (c *Controller) updateReleaseURL() string {
	if strings.TrimSpace(c.update.release.URL) != "" {
		return c.update.release.URL
	}
	return c.project.ReleasesURL
}

func updateCheckedLabel(value time.Time) string {
	if value.IsZero() {
		return "尚未检查"
	}
	return "检查于 " + value.Local().Format("01-02 15:04")
}
