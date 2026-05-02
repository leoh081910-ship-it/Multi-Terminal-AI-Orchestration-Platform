//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	if err := entc.Generate("./ent/schema", &gen.Config{
		Target: "./ent",
	}); err != nil {
		log.Fatalf("running entc generate: %v", err)
	}
}
