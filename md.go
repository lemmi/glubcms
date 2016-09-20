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
