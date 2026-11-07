package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"myjson/pkg/icons/generator"
)

func main() {
	if err := generator.Run(context.Background(), os.Args[1:], http.DefaultClient, os.Stdout); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, "icongen:", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
