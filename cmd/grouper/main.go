package main

import (
	"context"
	"log"

	"github.com/letieu/idea-extractor/internal/group"
)

func main() {
	grouper, err := group.New()
	if err != nil {
		log.Fatalf("fail to init grouper %v", err)
	}

	err = grouper.ProcessSourceItems(context.Background())
	if err != nil {
		log.Fatalf("fail to run group %v", err)
	}

	log.Printf("DONE")
}
