package core

import (
	"errors"
	"io"
	"runtime"
)

const networkNamespaceHolderCommand = "--boxd-netns-holder"

func networkNamespaceHolderArgs() []string {
	if runtime.GOOS != "linux" {
		return nil
	}
	return []string{"/proc/self/exe", networkNamespaceHolderCommand}
}

// RunInternalCommand 处理内核辅助进程；应在服务或桌面初始化之前调用。
func RunInternalCommand(args []string, input io.Reader) (bool, error) {
	if len(args) == 0 || args[0] != networkNamespaceHolderCommand {
		return false, nil
	}
	if len(args) != 1 {
		return true, errors.New("network namespace holder does not accept arguments")
	}
	if runtime.GOOS != "linux" {
		return true, errors.New("network namespaces require Linux")
	}
	_, err := io.Copy(io.Discard, input)
	return true, err
}
