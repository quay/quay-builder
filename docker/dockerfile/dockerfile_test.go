package dockerfile

import (
	"bytes"
	"testing"
)

func TestNewMetadataFailures(t *testing.T) {
	var table = []struct {
		dockerfile           string
		expectedErr          error
		expectedErrorMessage string
	}{
		{"", ErrEmptyDockerfile, ""},
		{"ADD . .", ErrDockerfileMissingFROMorARG, ""},
		{"FROM /invalid", ErrInvalidBaseImage, ""},
	}

	for _, tt := range table {
		t.Run(tt.dockerfile, func(t *testing.T) {
			_, err := NewMetadataFromReader(bytes.NewBufferString(tt.dockerfile), ".")
			if tt.expectedErrorMessage == "" {
				if err != tt.expectedErr {
					t.Fatalf("unexpected error: got: %s wanted: %s", err, tt.expectedErr)
				}
			} else {
				if err.Error() != tt.expectedErrorMessage {
					t.Fatalf("unexpected error: got: %s wanted: %s", err, tt.expectedErr)
				}
			}
		})
	}
}

func TestValidParsing(t *testing.T) {
	var table = []struct {
		dockerfile       string
		expectedMetadata *Metadata
	}{
		// Original format.
		{
			"FROM jzelinskie:image",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},
		{
			"FROM jzelinskie:image\nRUN foo",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},
		{
			"FROM jzelinskie:image\nRUN foo\nADD . .",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},

		// ADD and COPY
		{
			"FROM jzelinskie:image\nADD moby.txt .\nCOPY moby.txt .",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},

		// Environment variable replacement.
		{
			"FROM jzelinskie:image\nENV foo=bar\nWORKDIR $foo",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},
		{
			"FROM jzelinskie:image\nENV foo=bar\nWORKDIR ${foo}",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},

		// Multi-from
		{
			"FROM jzelinskie:image\nRUN foo\nFROM other:thing\nRUN bar",
			&Metadata{
				BaseImage:    "jzelinskie",
				BaseImageTag: "image",
			},
		},

		// Port in EXPOSE
		{
			"FROM ubuntu:latest\nENV PORT 3000\nEXPOSE $PORT",
			&Metadata{
				BaseImage:    "ubuntu",
				BaseImageTag: "latest",
			},
		},
	}

	for _, tt := range table {
		m, err := NewMetadataFromReader(bytes.NewBufferString(tt.dockerfile), "testdata")
		if err != nil {
			t.Fatalf("unexpected error: got: %s", err)
			continue
		}

		if m.BaseImage != tt.expectedMetadata.BaseImage {
			t.Fatalf("unexpected metadata: got: %s. expected: %s", m, tt.expectedMetadata)
			continue
		}

		if m.BaseImageTag != tt.expectedMetadata.BaseImageTag {
			t.Fatalf("unexpected metadata: got: %s. expected: %s", m, tt.expectedMetadata)
			continue
		}
	}
}
