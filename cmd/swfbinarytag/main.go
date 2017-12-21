package main

import (
	"flag"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/soh335/swfbinarytag"
)

var (
	inputName  = flag.String("input", "", "path to swf file")
	id         = flag.Int("id", 0, "binary id")
	outputName = flag.String("output", "", "path to output")
)

func main() {
	flag.Parse()
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	input, err := os.Open(*inputName)
	if err != nil {
		return errors.Wrapf(err, "failed to open file:%s", *inputName)
	}
	defer input.Close()

	data, err := swfbinarytag.Find(input, uint16(*id))
	if err != nil {
		return errors.Wrapf(err, "failed to find id:%d", *id)
	}

	output, err := os.Create(*outputName)
	if err != nil {
		return errors.Wrapf(err, "failed to open file:%s", *outputName)
	}

	defer output.Close()
	if _, err := output.Write(data); err != nil {
		return errors.Wrapf(err, "failed to write file:%s", *outputName)
	}

	return nil
}
