package core

import (
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"
)

func TestRunInternalCommandIgnoresNormalArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"--version"}, {"--listen", "127.0.0.1:9091"}} {
		handled, err := RunInternalCommand(args, strings.NewReader(""))
		if handled || err != nil {
			t.Fatalf("normal args were intercepted: %v: %v", args, err)
		}
	}
}

func TestRunInternalCommandRejectsExtraArguments(t *testing.T) {
	handled, err := RunInternalCommand([]string{networkNamespaceHolderCommand, "extra"}, strings.NewReader(""))
	if !handled || err == nil {
		t.Fatalf("expected holder argument error, handled=%v err=%v", handled, err)
	}
}

func TestRunInternalCommandReadsUntilEOF(t *testing.T) {
	handled, err := RunInternalCommand([]string{networkNamespaceHolderCommand}, strings.NewReader("hold"))
	if !handled || runtime.GOOS == "linux" && err != nil || runtime.GOOS != "linux" && err == nil {
		t.Fatalf("unexpected holder result: handled=%v err=%v", handled, err)
	}
}

func TestRunInternalCommandPropagatesReadFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network namespace holder requires Linux")
	}
	expected := errors.New("input failed")
	handled, err := RunInternalCommand([]string{networkNamespaceHolderCommand}, holderErrorReader{expected})
	if !handled || !errors.Is(err, expected) {
		t.Fatalf("expected input error, handled=%v err=%v", handled, err)
	}
}

type holderErrorReader struct{ err error }

func (r holderErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestNetworkNamespaceHolderUsesCurrentExecutable(t *testing.T) {
	args := networkNamespaceHolderArgs()
	if runtime.GOOS != "linux" {
		if args != nil {
			t.Fatal("non-Linux runtime should not configure a namespace holder")
		}
		return
	}
	if len(args) != 2 || args[0] != "/proc/self/exe" || args[1] != networkNamespaceHolderCommand {
		t.Fatalf("unexpected holder invocation: %v", args)
	}
}

func TestRunInternalCommandWaitsForParentPipeClose(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network namespace holder requires Linux")
	}
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	result := make(chan error, 1)
	go func() {
		_, err := RunInternalCommand([]string{networkNamespaceHolderCommand}, reader)
		result <- err
	}()
	if _, err := writer.Write([]byte("alive")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		t.Fatalf("holder exited before parent closed pipe: %v", err)
	default:
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
