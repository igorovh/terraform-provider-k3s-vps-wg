package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/igorovh/terraform-provider-k3s-vps-wg/internal/provider"
)

var version = "0.1.0"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/igorovh/k3s-vps-wg",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err)
	}
}
