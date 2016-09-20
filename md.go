package glubcms

import (
	"bytes"

	bf "github.com/russross/blackfriday"
)

type mdModifier struct {
	bf.Renderer
	printHeader bool
}

func NewMdModifier(r bf.Renderer) bf.Renderer {
	return &mdModifier{
		Renderer: r,
	}
}

func (md *mdModifier) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	if title == nil {
		title = alt
	}
	if alt == nil {
		alt = title
	}
	md.Renderer.Image(out, link, title, alt)
}

func (md *mdModifier) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	if level < 6 {
		level++
	}
	md.Renderer.Header(out, text, level, id)
}

func (md *mdModifier) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {
	// doubleSpace from blackfriday/html.go L897
	if out.Len() > 0 {
		out.WriteByte('\n')
	}

	out.WriteString("<table>\n")
	if md.printHeader {
		out.WriteString("<thead>\n")
		out.Write(header)
		out.WriteString("</thead>\n\n")
	}
	out.WriteString("<tbody>\n")
	out.Write(body)
	out.WriteString("</tbody>\n</table>\n")
}

func (md *mdModifier) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
	if len(text) > 0 {
		md.printHeader = true
	}
	md.Renderer.TableHeaderCell(out, text, align)
}
