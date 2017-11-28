package main

import (
	"fmt"
	"os"

	"gitlab.com/thedahv/pngaling/png"
)

func main() {
	var parsers []*png.Parser
	args := os.Args[1:]

	if len(args) == 0 {
		parsers = append(parsers, png.New("stdin", os.Stdin))
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
		b, err := p.IsPNG()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to check PNG header: %v", err)
			continue
		}

		fmt.Printf("PNG Status %s: %v\n", p.Path, b)

		err = p.Parse()
		if err != nil {
			fmt.Fprintf(os.Stderr, "problem parsing %s: %v", p.Path, err)
			continue
		}

		p.PrintHeader()
	}
}
