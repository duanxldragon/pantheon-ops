package contracts

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var runtimeSettingReloaders = struct {
	sync.RWMutex
	items map[string]func() error
}{
	items: make(map[string]func() error),
}

func RegisterRuntimeSettingReloader(name string, reload func() error) func() {
	name = strings.TrimSpace(name)
	if name == "" || reload == nil {
		return func() {}
	}

	runtimeSettingReloaders.Lock()
	runtimeSettingReloaders.items[name] = reload
	runtimeSettingReloaders.Unlock()

	return func() {
		runtimeSettingReloaders.Lock()
		delete(runtimeSettingReloaders.items, name)
		runtimeSettingReloaders.Unlock()
	}
}

func NotifyRuntimeSettingsChanged() error {
	runtimeSettingReloaders.RLock()
	names := make([]string, 0, len(runtimeSettingReloaders.items))
	for name := range runtimeSettingReloaders.items {
		names = append(names, name)
	}
	sort.Strings(names)

	reloaders := make([]func() error, 0, len(names))
	for _, name := range names {
		reloaders = append(reloaders, runtimeSettingReloaders.items[name])
	}
	runtimeSettingReloaders.RUnlock()

	errs := make([]string, 0)
	for index, reload := range reloaders {
		if err := reload(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", names[index], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("runtime settings reload failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
