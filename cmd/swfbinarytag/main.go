package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
		return fmt.Errorf("failed to open file:%s error:%s", *inputName, err.Error())
	}
	defer input.Close()

	data, err := swfbinarytag.Find(input, uint16(*id))
	if err != nil {
		return fmt.Errorf("failed to find id:%d error:%s", *id, err.Error())
	}

	output, err := os.Create(*outputName)
	if err != nil {
		return fmt.Errorf("failed to open file:%s error:%s", *outputName, err.Error())
	}

	defer output.Close()
	if _, err := output.Write(data); err != nil {
		return fmt.Errorf("failed to write file:%s error:%s", *outputName, err.Error())
	}

	return nil
}
