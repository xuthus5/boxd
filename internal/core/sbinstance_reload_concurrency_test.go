package core

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestSBInstanceConcurrentReloadCannotUndoStop(t *testing.T) {
	const attempts = 32
	const concurrentReloads = 8
	fixture := newSBReloadFixture(t)
	for range attempts {
		fixture.start(t)
		operations := []func() error{fixture.instance.Stop}
		for range concurrentReloads {
			operations = append(operations, fixture.instance.Reload)
		}
		runSBInstanceOperations(t, operations)
		if fixture.instance.Status().Running {
			t.Fatal("concurrent reload must not undo a stop request")
		}
	}
}

func TestSBInstanceConcurrentControlsReleaseResources(t *testing.T) {
	const concurrentControls = 16
	fixture := newSBReloadFixture(t)
	operations := make([]func() error, 0, concurrentControls)
	controls := []func() error{
		fixture.instance.Start, fixture.instance.Restart, fixture.instance.Reload, fixture.instance.Stop,
	}
	for index := range concurrentControls {
		operations = append(operations, controls[index%len(controls)])
	}
	runSBInstanceOperations(t, operations)
	if err := fixture.instance.Stop(); err != nil {
		t.Fatal(err)
	}
	for index, instance := range fixture.boxes {
		if !instance.closed || !errors.Is(fixture.options[index].Context.Err(), context.Canceled) {
			t.Fatalf("kernel %d leaked resources after concurrent controls", index)
		}
	}
}

func runSBInstanceOperations(t *testing.T, operations []func() error) {
	t.Helper()
	var workers sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, len(operations))
	for _, operation := range operations {
		workers.Go(func() {
			<-start
			results <- operation()
		})
	}
	close(start)
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
}
