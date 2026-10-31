package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/igorovh/terraform-provider-anpi/internal/provider"
)

var version = "0.2.0"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/igorovh/anpi",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
