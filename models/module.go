package models

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/pion/mediadevices/pkg/driver"
	mdcam "github.com/pion/mediadevices/pkg/driver/camera"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

var (
	WebcamDiscovery = resource.NewModel("rand", "find-webcams", "webcam-discovery")
)

func init() {
	resource.RegisterService(discovery.API, WebcamDiscovery,
		resource.Registration[discovery.Service, resource.NoNativeConfig]{
			Constructor: newFindWebcamsWebcamDiscovery,
		},
	)
}

type findCams struct {
	resource.Named
	resource.TriviallyCloseable
	resource.AlwaysRebuild

	logger logging.Logger
}

func newFindWebcamsWebcamDiscovery(_ context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger) (discovery.Service, error) {
	s := &findCams{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}
	return s, nil
}

// DiscoverResources implements discovery.Service.
func (s *findCams) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	return findCameras(ctx, getVideoDrivers, s.logger)
}

// getVideoDrivers is a helper callback passed to the registered Discover func to get all video drivers.
func getVideoDrivers() []driver.Driver {
	return driver.GetManager().Query(driver.FilterVideoRecorder())
}

// getProperties is a helper func for webcam discovery that returns the Media properties of a specific driver.
// It is NOT related to the GetProperties camera proto API.
func getProperties(d driver.Driver) (_ []prop.Media, err error) {
	// Need to open driver to get properties
	if d.Status() == driver.StateClosed {
		errOpen := d.Open()
		if errOpen != nil {
			return nil, errOpen
		}
		defer func() {
			if errClose := d.Close(); errClose != nil {
				err = errClose
			}
		}()
	}
	return d.Properties(), err
}

func fixName(name string) string {
	// First replace semicolons with hyphens
	name = strings.ReplaceAll(name, ";", "-")
	// remove all non-alphanumeric characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return reg.ReplaceAllString(name, "")
}

// Discover webcam attributes.
func findCameras(ctx context.Context, getDrivers func() []driver.Driver, logger logging.Logger) ([]resource.Config, error) {
	// Clear all registered camera devices before calling Initialize to prevent duplicates.
	// If first initalize call, this will be a noop.
	manager := driver.GetManager()
	for _, d := range manager.Query(driver.FilterVideoRecorder()) {
		manager.Delete(d.ID())
	}
	mdcam.Initialize()

	webcams := []resource.Config{}

	drivers := getDrivers()
	for _, d := range drivers {
		driverInfo := d.Info()

		props, err := getProperties(d)
		if len(props) == 0 {
			logger.CDebugw(ctx, "no properties detected for driver, skipping discovery...", "driver", driverInfo.Label)
			continue
		} else if err != nil {
			logger.CDebugw(ctx, "cannot access driver properties, skipping discovery...", "driver", driverInfo.Label, "error", err)
			continue
		}

		if d.Status() == driver.StateRunning {
			logger.CDebugw(ctx, "driver is in use, skipping discovery...", "driver", driverInfo.Label)
			continue
		}

		labelParts := strings.Split(driverInfo.Label, mdcam.LabelSeparator)

		// for mac, the device path is the first part of the label
		label := labelParts[0]
		if len(labelParts) > 1 {
			// for linux, the device path that works is the second part of the label
			// unknown why
			label = labelParts[1]
		}

		logger.Debugf("found camera drivers with info  %#v", driverInfo)

		var result map[string]interface{}
		attributes := videosource.WebcamConfig{
			Path: label,
			// TODO: find a way to return good properties without returning all combinations
			// Format: string(prop.Video.FrameFormat),
			// Width:     prop.Video.Width,
			// Height:    prop.Video.Height,
			// FrameRate: prop.Video.FrameRate,
		}

		// marshal to bytes
		jsonBytes, err := json.Marshal(attributes)
		if err != nil {
			return nil, err
		}

		// convert to map to be used as attributes in resource.Config
		if err = json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, err
		}

		name := fixName(driverInfo.Name)

		wc := resource.Config{
			Name:                name,
			API:                 camera.API,
			Model:               videosource.ModelWebcam,
			Attributes:          result,
			ConvertedAttributes: attributes,
		}

		webcams = append(webcams, wc)
	}

	return webcams, nil
}
