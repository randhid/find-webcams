package models

import (
	"context"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
)

const someName = "USB223;44 Somecam^"

// fakeDriver is a driver has a label and media properties.
type fakeDriver struct {
	label string
	props []prop.Media
}

func (d *fakeDriver) Open() error              { return nil }
func (d *fakeDriver) Properties() []prop.Media { return d.props }
func (d *fakeDriver) ID() string               { return d.label }
func (d *fakeDriver) Info() driver.Info        { return driver.Info{Label: d.label, Name: someName} }
func (d *fakeDriver) Status() driver.State     { return "some state" }
func (d *fakeDriver) Close() error             { return nil }

func newFakeDriver(label string, props []prop.Media) driver.Driver {
	return &fakeDriver{label: label, props: props}
}

func testGetDrivers() []driver.Driver {
	props := prop.Media{
		Video:    prop.Video{Width: 320, Height: 240, FrameFormat: "some format", FrameRate: 30.0},
		DeviceID: "some_device_id;",
	}
	withProps := newFakeDriver("some_label", []prop.Media{props})
	withoutProps := newFakeDriver("another label", []prop.Media{})
	return []driver.Driver{withProps, withoutProps}
}

func TestDiscoveryWebcam(t *testing.T) {
	logger := logging.NewTestLogger(t)
	resp, err := findCameras(context.Background(), testGetDrivers, logger)

	test.That(t, err, test.ShouldBeNil)
	test.That(t, resp, test.ShouldHaveLength, 1)
	test.That(t, resp[0].API, test.ShouldResemble, camera.API)
	test.That(t, resp[0].Name, test.ShouldResemble, fixName(someName))

	cfg, ok := resp[0].ConvertedAttributes.(videosource.WebcamConfig)
	test.That(t, ok, test.ShouldBeTrue)

	test.That(t, cfg, test.ShouldHaveSameTypeAs, videosource.WebcamConfig{})

	// these will need to be adde dback when the cfg popualtes them again
	test.That(t, cfg.Width, test.ShouldEqual, 0)
	test.That(t, cfg.Height, test.ShouldEqual, 0)
	test.That(t, cfg.Format, test.ShouldResemble, "")
	test.That(t, cfg.FrameRate, test.ShouldEqual, 0)
}
