// Package watch reports changes to the issue files in a directory, such as
// edits made in another editor, or by git, and to the state of the git
// repository they're in, such as commits.
package watch

import (
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// delay is how long a Watcher waits after a change for more, so a burst of
// changes, such as a git pull's, is reported once.
var delay = 100 * time.Millisecond

// Watcher reports changes to files: the issue files in a directory, or one
// file, such as a git reflog. A change is reported delay after the last of a
// burst of changes, and changes that haven't been received yet are reported
// together, so a receiver that's busy doesn't fall behind.
type Watcher struct {
	watcher  *fsnotify.Watcher
	relevant func(fsnotify.Event) bool
	changes  chan struct{}
	done     chan struct{}
}

// New returns a Watcher for the issue files in dir. It must be closed.
func New(dir string) (*Watcher, error) {
	return newWatcher(dir, issueFile)
}

// NewFile returns a Watcher for the file at path, which reports when it's
// created, written, removed, or replaced. The file needn't exist, but its
// directory must. It must be closed.
func NewFile(path string) (*Watcher, error) {
	path = filepath.Clean(path)

	return newWatcher(filepath.Dir(path), func(event fsnotify.Event) bool {
		return event.Op != fsnotify.Chmod && filepath.Clean(event.Name) == path
	})
}

// Reflog returns the path of HEAD's reflog in the git repository dir is in,
// which git adds a line to whenever HEAD moves: on a commit, pull, merge,
// rebase, checkout, or reset. In a worktree, it's the worktree's own. It
// returns an error if dir isn't in a git repository, or git isn't
// installed. The reflog may not exist yet, in a repository without commits,
// or one where it's turned off.
func Reflog(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--git-path", "logs/HEAD").Output()
	if err != nil {
		return "", err
	}

	// The path is relative to dir, unless it's outside the repository's
	// working tree, as a worktree's is.
	path := strings.TrimSpace(string(out))

	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}

	return filepath.Abs(path)
}

// newWatcher returns a Watcher for the files in dir that relevant reports
// changes to.
func newWatcher(dir string, relevant func(fsnotify.Event) bool) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := fs.Add(dir); err != nil {
		fs.Close()
		return nil, err
	}

	w := &Watcher{
		watcher:  fs,
		relevant: relevant,
		changes:  make(chan struct{}, 1),
		done:     make(chan struct{}),
	}

	go w.run()

	return w, nil
}

// Changes returns the channel a value is sent on when the watched files have
// been added, changed, or removed. It's closed when the Watcher is.
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

			if w.relevant(event) {
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

// issueFile reports whether event is a change to an issue file. The metadata
// database, the configuration, and editors' temporary files are ignored, as
// are changes to files' permissions.
func issueFile(event fsnotify.Event) bool {
	if event.Op == fsnotify.Chmod {
		return false
	}

	return filepath.Ext(event.Name) == ".md"
}
