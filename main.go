package main

import "flag"

var input = flag.String("input", ".", "Input directory")
var output = flag.String("output", ".", "Output directory")

func main() {
	flag.Parse()
	generate(*input, *output)
}
