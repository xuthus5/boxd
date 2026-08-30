//go:build linux && cgo

package main

/*
#cgo pkg-config: gtk4 webkitgtk-6.0

#include <gtk/gtk.h>
#include <webkit/webkit.h>

extern void boxdOnWebProcessTerminated(void);

static void on_web_process_terminated(WebKitWebView *webview,
                                      WebKitWebProcessTerminationReason reason,
                                      gpointer user_data) {
	(void)webview;
	(void)reason;
	(void)user_data;
	boxdOnWebProcessTerminated();
}

// connect_if_webview 命中 WebKitWebView 时挂接渲染进程终止信号，
// 并继续递归遍历子 widget（GTK4 无 gtk_container_foreach）。
static void connect_if_webview(GtkWidget *widget) {
	if (widget == NULL) {
		return;
	}
	if (WEBKIT_IS_WEB_VIEW(widget)) {
		g_signal_connect(widget, "web-process-terminated",
		                 G_CALLBACK(on_web_process_terminated), NULL);
	}
	for (GtkWidget *child = gtk_widget_get_first_child(widget); child != NULL;
	     child = gtk_widget_get_next_sibling(child)) {
		connect_if_webview(child);
	}
}

// connect_web_process_terminated_toplevels 遍历所有 GTK 顶层窗口，
// 返回挂接信号的 WebKitWebView 数量。
static gint connect_web_process_terminated_toplevels(void) {
	gint connected = 0;
	GList *toplevels = gtk_window_list_toplevels();
	for (GList *l = toplevels; l != NULL; l = l->next) {
		if (WEBKIT_IS_WEB_VIEW(GTK_WIDGET(l->data))) {
			connected++;
		}
		connect_if_webview(GTK_WIDGET(l->data));
	}
	g_list_free(toplevels);
	return connected;
}
*/
import "C"

import (
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	hookOnce sync.Once
	hookFn   func()
)

// hookWebProcessTerminated 在主窗口的 WebKitWebView 上挂接
// web-process-terminated 信号（幂等）；必须在 webview 创建后调用，
// 例如 WindowRuntimeReady 事件回调里。
func hookWebProcessTerminated(reloadFn func()) {
	hookOnce.Do(func() {
		hookFn = reloadFn
		var connected C.gint
		application.InvokeSync(func() {
			connected = C.connect_web_process_terminated_toplevels()
		})
		log.Printf("web process recovery hooked %d webview(s)", int(connected))
	})
}

// boxdOnWebProcessTerminated 由 C 侧信号回调调用，处于 GTK 主线程。
//
//export boxdOnWebProcessTerminated
func boxdOnWebProcessTerminated() {
	webRecovery.onTerminated()
}
