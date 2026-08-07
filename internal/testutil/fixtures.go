package testutil

import (
	"bytes"
	_ "embed"
	"fmt"
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

// MinimalPDFBytes returns a small PDF fixture for document content blocks.
// Built as raw bytes (not a UTF-8 string) so the PDF binary comment and xref
// offsets stay consistent for real parsers.
func MinimalPDFBytes() []byte {
	var b bytes.Buffer
	// Header + binary comment (four high bytes; must not go through UTF-8 runes).
	b.WriteString("%PDF-1.1\n%")
	b.Write([]byte{0xe2, 0xe3, 0xcf, 0xd3})
	b.WriteByte('\n')

	offsets := make([]int, 4)
	writeObj := func(n int, body string) {
		offsets[n] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>")

	xrefPos := b.Len()
	fmt.Fprintf(&b, "xref\n0 4\n")
	fmt.Fprintf(&b, "0000000000 65535 f \n")
	for i := 1; i <= 3; i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefPos)
	return b.Bytes()
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
