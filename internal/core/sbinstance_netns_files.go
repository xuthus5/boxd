package core

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type namespacePIDFile struct {
	tag  string
	path string
	file *os.File
	info os.FileInfo
}

type namespacePIDFiles []namespacePIDFile

var errNamespacePIDReplaced = errors.New("network namespace pid_file changed during startup")

func takeNamespacePIDFiles(namespaces []option.NetworkNamespace) ([]option.NetworkNamespace, namespacePIDFiles) {
	options := slices.Clone(namespaces)
	var files namespacePIDFiles
	for index := range options {
		namespace := &options[index]
		if namespace.Type != C.NetNsTypeUnshare || namespace.UnshareOptions.PidFile == "" {
			continue
		}
		files = append(files, namespacePIDFile{tag: namespace.Tag, path: namespace.UnshareOptions.PidFile})
		// 上游会以 0644 写入并按路径删除；由 boxd 保持权限与文件所有权。
		namespace.UnshareOptions.PidFile = ""
	}
	return options, files
}

func (files namespacePIDFiles) prepare() error {
	for index := range files {
		if err := files[index].create(); err != nil {
			return errors.Join(err, files.close())
		}
	}
	return nil
}

func (p *namespacePIDFile) create() error {
	file, err := os.OpenFile(p.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("prepare network namespace[%s] pid_file: %w", p.tag, err)
	}
	info, err := file.Stat()
	if err != nil {
		return errors.Join(err, file.Close(), os.Remove(p.path))
	}
	p.file, p.info = file, info
	return nil
}

func (files namespacePIDFiles) publish(manager adapter.NetworkNamespaceManager) error {
	for index := range files {
		if err := files[index].publish(manager); err != nil {
			return err
		}
	}
	return nil
}

func (p *namespacePIDFile) publish(manager adapter.NetworkNamespaceManager) error {
	if manager == nil || p.file == nil {
		return fmt.Errorf("network namespace[%s] pid_file was not initialized", p.tag)
	}
	if err := p.verifyPath(); err != nil {
		return err
	}
	path := manager.ResolvePath(p.tag)
	pid, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(path, "/proc/"), "/ns/net"))
	if err != nil || pid <= 0 || path != fmt.Sprintf("/proc/%d/ns/net", pid) {
		return fmt.Errorf("network namespace[%s] returned an invalid holder path", p.tag)
	}
	_, writeErr := p.file.WriteString(strconv.Itoa(pid) + "\n")
	if writeErr == nil {
		writeErr = p.file.Sync()
	}
	closeErr := p.file.Close()
	p.file = nil
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("write network namespace[%s] pid_file: %w", p.tag, err)
	}
	return p.verifyPath()
}

func (p *namespacePIDFile) verifyPath() error {
	info, err := os.Lstat(p.path)
	if err != nil {
		return fmt.Errorf("inspect network namespace[%s] pid_file: %w", p.tag, err)
	}
	if p.info == nil || !info.Mode().IsRegular() || !os.SameFile(p.info, info) || info.Mode().Perm() != 0600 {
		return fmt.Errorf("%w: %s", errNamespacePIDReplaced, p.path)
	}
	return nil
}

func (files namespacePIDFiles) close() error {
	var err error
	for index := range files {
		err = errors.Join(err, files[index].close())
	}
	return err
}

func (p *namespacePIDFile) close() error {
	var err error
	if p.file != nil {
		err = p.file.Close()
		p.file = nil
	}
	if p.info == nil {
		return err
	}
	info, statErr := os.Lstat(p.path)
	if errors.Is(statErr, os.ErrNotExist) {
		p.info = nil
		return err
	}
	if statErr != nil {
		return errors.Join(err, statErr)
	}
	if !os.SameFile(p.info, info) {
		p.info = nil
		return err
	}
	removeErr := os.Remove(p.path)
	if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		p.info = nil
	}
	return errors.Join(err, removeErr)
}
