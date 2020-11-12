package stdforward

import (
	"fmt"
	"io"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

// stdforward will forward to any number of writers the messages on Stdout and StdErr.
// contrary to multiwriters, the list can go and shrink dynamically.

var (
	stdoutForwarder, stderrForwarder forwarder
)

type forwarder struct {
	out     *os.File
	writers map[string]io.Writer
	mu      sync.RWMutex

	once sync.Once
}

func (f *forwarder) Write(p []byte) (int, error) {
	// Write to regular output first
	n, err := f.out.Write(p)
	if err != nil {
		log.Warningf("Failed to write to regular output: %v", err)
	}

	// Now, forward to any registered writers
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, w := range f.writers {
		if _, err := w.Write(p); err != nil {
			log.Warningf("Failed to forward logs: %v", err)
		}
	}

	return n, nil
}

// AddStdoutWriter will forward stdout to writer (and all previous writers).
// First call switch Stdout to intercept any calls and forward it. Anything that
// referenced beforehand os.Stdout directly and captured it will thus
// not be forwarded.
func AddStdoutWriter(id string, w io.Writer) (fnErr error) {
	return addWriter(&stdoutForwarder, os.Stdout, id, w)
}

// RemoveStdoutWriter remove current id from stdout redirections.
func RemoveStdoutWriter(id string) {
	stdoutForwarder.mu.Lock()
	defer stdoutForwarder.mu.Unlock()
	delete(stdoutForwarder.writers, id)
}

// AddStderrWriter will forward stderr to writer (and all previous writers).
// First call switch Stderr to intercept any calls and forward it. Anything that
// referenced beforehand os.Stderr directly and captured it will thus
// not be forwarded.
func AddStderrWriter(id string, w io.Writer) (fnErr error) {
	return addWriter(&stderrForwarder, os.Stderr, id, w)
}

// RemoveStderrWriter remove current id from stderr redirections.
func RemoveStderrWriter(id string) {
	stderrForwarder.mu.Lock()
	defer stderrForwarder.mu.Unlock()
	delete(stderrForwarder.writers, id)
}

func addWriter(dest *forwarder, std *os.File, id string, w io.Writer) error {
	// Initialize our forwarder
	var onceErr error
	dest.once.Do(func() {
		dest.out = std
		dest.writers = make(map[string]io.Writer)

		rOut, wOut, err := os.Pipe()
		if err != nil {
			onceErr = fmt.Errorf("Can't redirect output: %v", err)
			return
		}

		go func() {
			if _, err = io.Copy(dest, rOut); err != nil {
				log.Warningf("Forwarding some messages failed: %v", err)
			}
		}()

		os.Stdout = wOut
	})
	if onceErr != nil {
		return onceErr
	}

	dest.mu.Lock()
	defer dest.mu.Unlock()
	dest.writers[id] = w

	return nil
}
