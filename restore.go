package mygo

import (
	"os"
	"path/filepath"
	"time"

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

// inDevelopment reports whether mygo dev runs the app, which it restarts as the
// code changes: the app comes back as it was, as hot restart keeps it.
func inDevelopment() bool { return os.Getenv("MYGO_DEV") != "" }

// keepStateInDevelopment keeps what the app gives every second, as mygo dev ends
// the app without a word.
func (a *Application) keepStateInDevelopment() {
	if !inDevelopment() || a.keepingState {
		return
	}
	a.keepingState = true
	a.WhenReady(func() {
		go func() {
			for range time.Tick(time.Second) {
				if !postMain(a.writeState) {
					return
				}
			}
		}()
	})
}

// writeState keeps what the app gives, on phones and under mygo dev.
func (a *Application) writeState() {
	if a.saveState == nil {
		return
	}
	if _, ok := backend().(platform.DeviceBackend); !ok && !inDevelopment() {
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
	if d, ok := backend().(interface{ WasEndedBySystem() bool }); ok && !d.WasEndedBySystem() && !inDevelopment() {
		os.Remove(path)
		return nil
	}
	if _, ok := backend().(platform.DeviceBackend); !ok && !inDevelopment() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	os.Remove(path)
	return data
}
