package glubcms

import (
	"bytes"

	bf "github.com/russross/blackfriday"
)

type ImageAltTitleCopy struct {
	bf.Renderer
}

func (md ImageAltTitleCopy) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	if title == nil {
		title = alt
	}
	if alt == nil {
		alt = title
	}
	md.Renderer.Image(out, link, title, alt)
}

type CorrectHeadingLevel struct {
	bf.Renderer
}

func (md CorrectHeadingLevel) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	if level < 6 {
		level++
	}
	md.Renderer.Header(out, text, level, id)
}
