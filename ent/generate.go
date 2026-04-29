//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/cmd/entc"
	"entgo.io/ent/cmd/entc/generate"
)

func main() {
	if err := entc.Generate("./schema", &generate.Config{
		Target: "./ent",
	}); err != nil {
		log.Fatalf("running entc generate: %v", err)
	}
}
