# PNGuin

A simple tool for viewing and cleaning the non-image metadata from PNG files.

## Overview

This is more a simple little exercise that was fun enough to share; consider
this an effort of learning more than utility.

If you didn't already know, your image files can contain more information than
just the picture they capture. There is potentially geolocation, image-editing
information, text comments, and more embedded in the image too.

I hadn't worked on a fun Go project in a while, and I was curious to know how
complicated the PNG spec was. It turns out, it isn't. The
[Wikipedia page](https://en.wikipedia.org/wiki/Portable_Network_Graphics) is
thorough and well-documented and this kind of spec translates nicely to Go
structs and utilities found in the [io](https://godoc.org/io) and
[binary](https://godoc.org/encoding/binary) packages.

This little tool gives you some insight into what is really in your PNG files
and lets you clean them up if you so desire.

## Usage

		./pnguin --help
		usage: pnguin [imgpath ...]
			-clean
						Write images stripped of text tags
			-tags
						Print non-data tags

`pnguin` can either take a list of paths to images to process, or can take PNG
file contents from stdin:

		$ curl -s https://upload.wikimedia.org/wikipedia/commons/3/39/PNG_demo_heatmap_Banana.png \
		| ./pnguin -tags
		stdin tags:
			PLTE (Pallette)

`pnguin` can also create copies of your images with all these tags removed. Its
naming convention is to either:

- Modify the end of the filename to `-cleaned.png` and save it in the folder of
  the original image
- Take files from stdin and name them `stdin-n.png` where `n` is the order of
  the input, starting with 0. Files are saved to the current working directory.
