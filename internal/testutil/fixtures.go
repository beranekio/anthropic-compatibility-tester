package testutil

import (
	"bytes"
	_ "embed"
	"io"
)

//go:embed testdata/small.png
var smallPNG []byte

const smallTextFileContent = "compatibility test file\n"

const smallSkillFileContent = `---
name: compatibility-test-skill
description: compatibility test skill
---

Compatibility test skill instructions.
`

const skillVersionUpdatedContent = `---
name: compatibility-test-skill
description: compatibility test skill v2
---

Compatibility test skill instructions v2.
`

// SkillBundleFolder is the top-level directory name in skill upload zip bundles.
const SkillBundleFolder = "compatibility-test-skill"

type namedPNGReader struct {
	r *bytes.Reader
}

func (r *namedPNGReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *namedPNGReader) Filename() string {
	return "test.png"
}

func (r *namedPNGReader) ContentType() string {
	return "image/png"
}

// SmallPNGBytes returns a copy of the embedded 8x8 RGBA PNG used for multipart image uploads.
func SmallPNGBytes() []byte {
	buf := make([]byte, len(smallPNG))
	copy(buf, smallPNG)
	return buf
}

// SmallPNGReader returns a multipart-ready reader for the embedded PNG fixture.
func SmallPNGReader() io.Reader {
	return &namedPNGReader{r: bytes.NewReader(smallPNG)}
}

type namedTextReader struct {
	r        *bytes.Reader
	filename string
}

func (r *namedTextReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *namedTextReader) Filename() string {
	return r.filename
}

func (r *namedTextReader) ContentType() string {
	return "text/plain"
}

// SmallTextFileReader returns a multipart-ready reader for a small text file fixture.
func SmallTextFileReader() io.Reader {
	return &namedTextReader{
		r:        bytes.NewReader([]byte(smallTextFileContent)),
		filename: "test.txt",
	}
}

// SmallTextFileBytes returns the bytes of the small text file fixture.
func SmallTextFileBytes() []byte {
	return []byte(smallTextFileContent)
}

type namedSkillFileReader struct {
	r        *bytes.Reader
	filename string
}

func (r *namedSkillFileReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *namedSkillFileReader) Filename() string {
	return r.filename
}

func (r *namedSkillFileReader) ContentType() string {
	return "text/markdown"
}

// SmallSkillFileReader returns a multipart-ready reader for a minimal skill bundle file.
func SmallSkillFileReader() io.Reader {
	return SkillFileReader(smallSkillFileContent)
}

// SkillVersionFileReader returns a multipart-ready reader for an updated skill bundle file.
func SkillVersionFileReader() io.Reader {
	return SkillFileReader(skillVersionUpdatedContent)
}

// SkillFileReader returns a multipart-ready reader for skill bundle content.
func SkillFileReader(content string) io.Reader {
	return &namedSkillFileReader{
		r:        bytes.NewReader([]byte(content)),
		filename: SkillBundleFolder + "/SKILL.md",
	}
}