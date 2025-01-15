package models

import (
	"context"
	"errors"
	"strings"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/rdk/utils"
)

var (
	WebcamDiscovery  = resource.NewModel("rand", "find-webcams", "webcam-discovery")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(discovery.API, WebcamDiscovery,
		resource.Registration[discovery.Service, *Config]{
			Constructor: newFindWebcamsWebcamDiscovery,
		},
	)
}

type Config struct {
	resource.TriviallyValidateConfig
}

type findWebcamsWebcamDiscovery struct {
	resource.Named
	resource.TriviallyCloseable
	resource.AlwaysRebuild

	logger logging.Logger
	cfg    *Config
}

func newFindWebcamsWebcamDiscovery(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (discovery.Service, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	s := &findWebcamsWebcamDiscovery{
		Named:  rawConf.ResourceName().AsNamed(),
		logger: logger,
		cfg:    conf,
	}
	return s, nil
}

// DiscoverResources implements discovery.Service.
func (s *findWebcamsWebcamDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
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

// Discover webcam attributes.
func findCameras(ctx context.Context, getDrivers func() []driver.Driver, logger logging.Logger) ([]resource.Config, error) {
	driver.Initialize()
	var webcams []resource.Config
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

		labelParts := strings.Split(driverInfo.Label, driver.LabelSeparator)
		label := labelParts[0]

		name, id := func() (string, string) {
			nameParts := strings.Split(driverInfo.Name, driver.LabelSeparator)
			if len(nameParts) > 1 {
				return nameParts[0], nameParts[1]
			}
			// fallback to the label if the name does not have an any additional parts to use.
			return nameParts[0], label
		}()

		for _, prop := range props {
			attributes := videosource.WebcamConfig{
				Path:      id,
				Format:    string(prop.Video.FrameFormat),
				Width:     prop.Video.Width,
				Height:    prop.Video.Height,
				FrameRate: prop.Video.FrameRate,
			}

			wc := resource.Config{
				Name:                name,
				API:                 camera.API,
				Model:               videosource.ModelWebcam,
				Attributes:          utils.AttributeMap{},
				ConvertedAttributes: attributes,
			}

			webcams = append(webcams, wc)
		}
	}
	return webcams, nil
}
