package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuthus5/boxd/internal/service"
)

func (a *application) openAddNode() {
	if a.backend == nil || a.model.busy {
		return
	}
	a.nodeForm.reset()
}

func (a *application) openEditNode(node nodeView) {
	if a.backend == nil || a.model.busy || node.Source != "import" {
		return
	}
	a.nodeForm.beginEdit(node)
}

func (a *application) submitNodeForm() {
	if a.backend == nil || a.model.busy {
		return
	}
	if !a.nodeForm.edit {
		link := strings.TrimSpace(a.nodeForm.link)
		if link == "" {
			a.nodeForm.err = "Proxy link is required"
			return
		}
		a.nodeForm.err = ""
		a.runActionWithCallback("Importing node", func(ctx context.Context) error {
			_, err := a.backend.ImportNode(ctx, link)
			return err
		}, "Node imported", func() {
			a.nodeForm.open = false
			a.nodeForm.link = ""
		})
		return
	}
	input, err := a.nodeForm.input()
	if err != nil {
		a.nodeForm.err = shortError(err)
		return
	}
	a.nodeForm.err = ""
	a.runActionWithCallback("Updating node", func(ctx context.Context) error {
		return a.backend.UpdateNode(ctx, a.nodeForm.editingTag, input)
	}, "Node updated", func() { a.nodeForm.open = false })
}

func (a *application) askDeleteNode(node nodeView) {
	if a.backend == nil || a.model.busy || node.Source != "import" {
		return
	}
	a.deleteForm = deleteFormState{
		open: true, kind: "node", id: node.Tag, title: "Delete node “" + node.Tag + "”?",
		message: "The imported node, its probe history and its managed outbound will be removed.", confirm: "Delete",
	}
}

func (a *application) deleteNode(tag string) {
	a.runActionWithCallback("Deleting node", func(ctx context.Context) error {
		return a.backend.DeleteNode(ctx, tag)
	}, "Node deleted", func() { a.requestRefresh() })
}

func (a *application) testNode(node nodeView) {
	if a.backend == nil || a.model.busy {
		return
	}
	req := service.TestRequest{Tag: node.Tag, TestType: "tcp", Server: node.Server, Port: node.Port}
	a.runAction("Testing node", func(ctx context.Context) error {
		result, err := a.backend.TestNode(ctx, req)
		if err != nil {
			return err
		}
		if !result.Success {
			return fmt.Errorf("node test failed: %s", result.Error)
		}
		return nil
	}, "Node test completed")
}

func (a *application) testAllNodes() {
	if a.backend == nil || a.model.busy || len(a.model.nodes) == 0 {
		return
	}
	items := make([]service.TestRequest, 0, min(len(a.model.nodes), 128))
	for _, node := range a.model.nodes {
		if len(items) >= 128 {
			break
		}
		items = append(items, service.TestRequest{Tag: node.Tag, TestType: "tcp", Server: node.Server, Port: node.Port})
	}
	a.runAction("Testing nodes", func(ctx context.Context) error {
		_, err := a.backend.TestNodes(ctx, service.TestBatchRequest{Items: items, Concurrency: 8})
		return err
	}, "Node tests completed")
}
