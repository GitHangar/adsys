package testutils

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	goCoverProfile   string
	coveragesToMerge []string
	onceCovFile      sync.Once
)

// AddCoverageFile append cov to the list of file to merge when calling MergeCoverages
func AddCoverageFile(cov string) {
	onceCovFile.Do(func() {
		goCoverProfile = testCoverageFile()
	})
	coveragesToMerge = append(coveragesToMerge, cov)
}

// MergeCoverages append all coverage files marked for merging to main Go Cover Profile
func MergeCoverages() {
	for _, cov := range coveragesToMerge {
		if err := appendToFile(goCoverProfile, cov); err != nil {
			log.Fatalf("can’t inject python coverage to golang one: %v", err)
		}
	}
}

// testCoverageFile returns the coverprofile file relative path.
// It returns nothing if coverage is not enabled.
func testCoverageFile() string {
	for _, arg := range os.Args {
		if !strings.HasPrefix(arg, "-test.coverprofile=") {
			continue
		}
		return strings.TrimPrefix(arg, "-test.coverprofile=")
	}
	return ""
}

// appendToFile appends toInclude to the coverprofile file at the end
func appendToFile(main, add string) error {
	d, err := os.ReadFile(add)
	if err != nil {
		return fmt.Errorf("can't open python coverage file named: %v", err)
	}

	f, err := os.OpenFile(main, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("can't open golang cover profile file: %v", err)
	}
	if _, err := f.Write(d); err != nil {
		return fmt.Errorf("can't write to golang cover profile file: %v", err)
	}
	return nil
}
