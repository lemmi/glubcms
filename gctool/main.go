package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/lemmi/glubcms"
)

func delspace(r rune) rune {
	if unicode.In(r, unicode.Latin, unicode.Digit) {
		return r
	} else {
		return '_'
	}
}

func main() {
	author := flag.String("author", "Webmaster", "Set the autorname")
	title := flag.String("title", "New Page", "Set the title")
	dirname := flag.String("dirname", "", "Set the directory name")
	priority := flag.Int("priority", 0, "Set the priority")
	simulate := flag.Bool("n", false, "Only show the result")
	flag.Parse()

	if *dirname == "" {
		*dirname = strings.Map(delspace, strings.ToLower(*title))
	}

	b, err := json.MarshalIndent(glubcms.Meta{
		Author:   *author,
		Title:    *title,
		Date:     glubcms.GCTime(time.Now()),
		Priority: *priority,
	}, "", "\t")

	if err != nil {
		panic(err)
	}

	if !*simulate {
		if err := os.Mkdir(*dirname, 0755); err != nil {
			panic(err)
		}

		metafile, err := os.Create(filepath.Join(*dirname, "meta.json"))
		if err != nil {
			//remove dir?
			panic(err)
		}
		defer metafile.Close()

		_, err = metafile.Write(b)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(*dirname)
	fmt.Println(string(b))
}
