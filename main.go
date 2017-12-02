package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab.com/thedahv/pnguin/png"
)

func main() {
	var parsers []*png.Parser

	showTags := flag.Bool("tags", false, "Print non-data tags")
	cleanFile := flag.Bool("clean", false,
		"Write images stripped of text tags")
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		parsers = append(parsers, png.New("Unknown Image", os.Stdin))
	} else {
		for _, path := range args {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to open files: %v", err)
				return
			}

			parsers = append(parsers, png.New(path, f))
		}
	}

	defer func() {
		for _, p := range parsers {
			p.Close()
		}
	}()

	for _, p := range parsers {
		if b, err := p.IsPNG(); !b || err != nil {
			fmt.Fprintf(os.Stderr, "%s is not a PNG\n", p.Path)
			continue
		}

		if err := p.Parse(); err != nil {
			fmt.Fprintf(os.Stderr, "problem parsing %s: %v", p.Path, err)
			continue
		}

		if *showTags {
			fmt.Fprintf(os.Stdout, "%s tags:\n", p.Path)
			p.WalkChunks(func(ch png.Chunk) bool {
				if !(ch.Type == png.ChunkTypeData || ch.Type == png.ChunkTypeHeader || ch.Type == png.ChunkTypeEnd) {
					fmt.Fprintf(os.Stdout, "  %s\n", ch.Type)
				}
				if ch.Type == png.ChunkTypeTxtUTF8 || ch.Type == png.ChunkTypeTxtISO8859 {
					fmt.Fprintf(os.Stdout, "   %s\n", ch.Data)
				}
				return true
			})
		}
	}
}
