package mygo

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo/internal/platform"
)

// What OnSaveState gives goes in a file of the app's, which the next launch
// reads and removes unless the host says it was not ended by the system.

func restorationFile() (string, error) {
	dir, err := App.Path(PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "restoration.state"), nil
}

// writeState keeps what the app gives, on phones.
func (a *Application) writeState() {
	if a.saveState == nil {
		return
	}
	if _, ok := backend().(platform.DeviceBackend); !ok {
		return
	}
	data := a.saveState()
	path, err := restorationFile()
	if err != nil {
		return
	}
	if len(data) == 0 {
		os.Remove(path)
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		os.Rename(tmp, path)
	}
}

// restoredState reads what was kept, once.
func restoredState() []byte {
	needsApp("App.RestoredState")
	path, err := restorationFile()
	if err != nil {
		return nil
	}
	if d, ok := backend().(interface{ WasEndedBySystem() bool }); ok && !d.WasEndedBySystem() {
		os.Remove(path)
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	os.Remove(path)
	return data
}
