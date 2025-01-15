package main

import (
	"find-webcams/models"

	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

func main() {
	module.ModularMain(resource.APIModel{discovery.API, models.WebcamDiscovery})
}
