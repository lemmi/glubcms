package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	hidden := flag.Bool("hidden", false, "Hide the page")
	edit := flag.Bool("e", false, "Open vim to edit the files")
	flag.Parse()

	meta := glubcms.Meta{
		Author:   *author,
		Title:    *title,
		Date:     glubcms.GCTime(time.Now()),
		Priority: *priority,
		Hidden:   *hidden,
	}

	if *dirname == "" {
		*dirname = time.Time(meta.Date).Format("2006-01-02_")
		*dirname += strings.NewReplacer(
			"ä", "ae",
			"ö", "oe",
			"ü", "ue",
			"ß", "ss").Replace(
			strings.Map(delspace, strings.ToLower(*title)))
	}

	b, err := json.MarshalIndent(meta, "", "\t")

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

		func(dirname string) {
			articlefile, err := os.Create(filepath.Join(dirname, "article.md"))
			if err != nil {
				panic(err)
			}
			defer articlefile.Close()
			if _, err := fmt.Fprintf(articlefile, "![title](/static/images/%s/0001.jpg)\n", dirname); err != nil {
				panic(err)
			}
			if _, err := fmt.Fprintf(articlefile, "*Foto: *\n"); err != nil {
				panic(err)
			}
		}(*dirname)
	}
	fmt.Println(*dirname)
	fmt.Println(string(b))

	if *edit {
		vimpath, err := exec.LookPath("vim")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Fount vim in %q\n", vimpath)
		cmd := exec.Command(
			vimpath,
			"-O",
			filepath.Join(*dirname, "article.md"),
			filepath.Join(*dirname, "meta.json"))
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(cmd.Run())
		}
	}
}
