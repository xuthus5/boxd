package core

import (
	"path/filepath"
	"sync"
)

var configOperationLocks sync.Map

// LockConfig 串行化同一配置的写入和节点同步，覆盖完整的读改写事务。
func LockConfig(path string) func() {
	key, err := filepath.Abs(path)
	if err != nil {
		key = filepath.Clean(path)
	}
	value, _ := configOperationLocks.LoadOrStore(key, &sync.Mutex{})
	lock, _ := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}
