module github.com/lemmi/glubcms

go 1.17

replace github.com/gogits/git => github.com/lemmi/git v0.0.2

require (
	github.com/gogits/git v0.0.0-00010101000000-000000000000
	github.com/lemmi/compress v0.0.0-20161005222315-481bb1aa824f
	github.com/lemmi/ghfs v0.0.0-20170601003624-6275da6ae931
	github.com/microcosm-cc/bluemonday v1.0.16
	github.com/pkg/errors v0.9.1
	github.com/raymondbutcher/tidyhtml v0.0.0-20150509150256-94c68d4f3550
	github.com/russross/blackfriday v1.6.0
)

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/unknwon/cae v1.0.2 // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
)
