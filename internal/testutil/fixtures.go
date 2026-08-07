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

// minimalPDF is a tiny PDF used for document/PDF message suites.
// It is intentionally small and only needs to be valid enough for SDK transport.
const minimalPDF = `%PDF-1.1
%âãÏÓ
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>
endobj
xref
0 4
0000000000 65535 f 
0000000015 00000 n 
0000000074 00000 n 
0000000133 00000 n 
trailer
<< /Size 4 /Root 1 0 R >>
startxref
210
%%EOF
`

// MinimalPDFBytes returns a small PDF fixture for document content blocks.
func MinimalPDFBytes() []byte {
	return []byte(minimalPDF)
}

// SkillVersionDownloadBytes is canned content returned by the mock skill version download.
func SkillVersionDownloadBytes() []byte {
	return []byte("PK\x03\x04mock-skill-version-content")
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
