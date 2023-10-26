package adsys_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocChapter(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")

	tests := map[string]struct {
		chapter string

		systemAnswer     string
		daemonNotStarted bool

		wantInDoc string
		wantErr   bool
	}{
		"Get documentation chapter":                     {chapter: "how-to-guides/set-up-ad", wantInDoc: "# How to set up the Active Directory Server"},
		"Get documentation chapter with incorrect case": {chapter: "HoW-to-guIdes/set-Up-AD", wantInDoc: "# How to set up the Active Directory Server"},

		// Section cases
		"Section using alias":                  {chapter: "how-to-guides", wantInDoc: "# How-to guides"},
		"Section using alias terminated by /":  {chapter: "how-to-guides/", wantInDoc: "# How-to guides"},
		"Section using title instead of alias": {chapter: "explanation", wantInDoc: "Scripts execution"},

		// Main index cases
		"Get main index with no parameter":         {wantInDoc: "# ADSys Documentation"},
		"Get main index with index title doc name": {chapter: "adsys-documentation", wantInDoc: "# ADSys Documentation"},

		"Get documentation is always authorized": {systemAnswer: "polkit_no", chapter: "how-to-guides/set-up-ad", wantInDoc: "# How to set up the Active Directory Server"},

		// Error cases
		"Error on daemon not responding": {daemonNotStarted: true, wantErr: true},
		"Error on nonexistent chapter":   {chapter: "nonexistent-chapter", wantErr: true},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.systemAnswer == "" {
				tc.systemAnswer = "polkit_yes"
			}
			dbusAnswer(t, tc.systemAnswer)

			conf := createConf(t)
			if !tc.daemonNotStarted {
				defer runDaemon(t, conf)()
			}

			args := []string{"doc"}
			if tc.chapter != "" {
				args = append(args, tc.chapter)
			}

			out, err := runClient(t, conf, args...)
			if tc.wantErr {
				require.Error(t, err, "client should exit with an error")
				return
			}

			require.NoError(t, err, "client should exit with no error")

			// Printing on stdout
			require.NotEmpty(t, out, "some documentation is printed")
			require.Contains(t, out, tc.wantInDoc, "Contains part of the expected doc content")

			// Note: (../images will be invalid when images are moved and this assertion will still be true
			assert.NotContains(t, out, "(../images/", "Local images are referenced, and replaced with online version")
		})
	}
}

func TestDocCompletion(t *testing.T) {
	tests := map[string]struct {
		systemAnswer     string
		daemonNotStarted bool

		wantCompletionEmpty bool
	}{
		"Completion lists main index, one section and one document": {},

		"Completion on documentation is always authorized": {systemAnswer: "polkit_no"},

		// Error cases
		"Empty completion content on daemon not responding": {daemonNotStarted: true, wantCompletionEmpty: true},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.systemAnswer == "" {
				tc.systemAnswer = "polkit_yes"
			}
			dbusAnswer(t, tc.systemAnswer)

			conf := createConf(t)
			if !tc.daemonNotStarted {
				defer runDaemon(t, conf)()
			}

			args := []string{"__complete", "doc", ""}
			out, err := runClient(t, conf, args...)
			require.NoError(t, err, "client should exit with no error")

			completions := strings.Split(out, "\n")

			if tc.wantCompletionEmpty {
				require.Len(t, completions, 2, "Should list no completion apart from :4 and empty")
				return
			}

			// Ensure that all interesting docs are listed here (and so. linked to their TOC)
			var wantNumDocs int
			docsDir := filepath.Join(rootProjectDir, "docs")
			err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
				// Ignore directories, every reference doc under reuse and compiled content.
				if d.IsDir() || strings.HasPrefix(path, filepath.Join(docsDir, "reuse")) || strings.HasPrefix(path, docsDir+"/.") {
					return nil
				}
				if !strings.HasSuffix(d.Name(), ".md") {
					return nil
				}
				wantNumDocs++

				return nil
			})
			require.NoError(t, err, "Setup: could not list existing doc on tests for tests to compare")

			// +2 as we have :4 for no file completion + empty string
			assert.Len(t, completions, wantNumDocs+2, "Should list all available documentation md files from docs/")

			assert.Contains(t, completions, "how-to-guides", "contain a section index")
			assert.Contains(t, completions, "how-to-guides/set-up-ad", "contain a section sub chapter with alias")
		})
	}
}
