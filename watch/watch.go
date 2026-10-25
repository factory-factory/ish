// Package watch reports changes to the issue files in a directory, such as
// edits made in another editor, or by git.
package watch

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// delay is how long a Watcher waits after a change for more, so a burst of
// changes, such as a git pull's, is reported once.
var delay = 100 * time.Millisecond

// Watcher reports changes to the issue files in a directory. A change is
// reported delay after the last of a burst of changes, and changes that
// haven't been received yet are reported together, so a receiver that's
// busy doesn't fall behind.
type Watcher struct {
	watcher *fsnotify.Watcher
	changes chan struct{}
	done    chan struct{}
}

// New returns a Watcher for the issue files in dir. It must be closed.
func New(dir string) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := fs.Add(dir); err != nil {
		fs.Close()
		return nil, err
	}

	w := &Watcher{
		watcher: fs,
		changes: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}

	go w.run()

	return w, nil
}

// Changes returns the channel a value is sent on when issue files have been
// added, changed, or removed. It's closed when the Watcher is.
func (w *Watcher) Changes() <-chan struct{} {
	return w.changes
}

// Close stops watching.
func (w *Watcher) Close() error {
	err := w.watcher.Close()
	<-w.done

	return err
}

func (w *Watcher) run() {
	defer close(w.done)
	defer close(w.changes)

	timer := time.NewTimer(delay)
	timer.Stop()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if relevant(event) {
				timer.Reset(delay)
			}

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}

			// Changes may have been missed, as when too many happen at once,
			// so the receiver should look at every file again.
			timer.Reset(delay)

		case <-timer.C:
			select {
			case w.changes <- struct{}{}:
			default:
				// A change is already waiting to be received.
			}
		}
	}
}

// relevant reports whether event is a change to an issue file. The metadata
// database, the configuration, and editors' temporary files are ignored, as
// are changes to files' permissions.
func relevant(event fsnotify.Event) bool {
	if event.Op == fsnotify.Chmod {
		return false
	}

	return filepath.Ext(event.Name) == ".md"
}
