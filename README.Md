# Webcam Discovery Module for Viam

This module provides a viam discovery service for automatically detecting and configuring webcams for use with the [Viam platform](https://www.viam.com).

## Overview

The [`webcam-discovery`](models/module.go) discovery service automatically:
- Scans for connected USB webcam devices
- Detects their capabilities and configuration
- Generates the appropriate Viam resource configurations

## Attributes
No models in this module have any attributes. Just add the model to your machine, expand the test card or look at the discovery control card, and copy/paste the video path to the `video_path` attribute on a `rdk:builtin:webcam` or click the add component model.

The discovery service detects the highest available resolution of the first format it finds for a usb webcam connected to the machine running viam-server.

## Example config

Add this module to your Viam robot's configuration:

```json
{
    {
  "services": [
    {
      "name": "webcam-discovery",
      "api": "rdk:service:discovery", 
      "model": "rand:find-webcams:webcam-discovery",
      "attributes" : {}
    }
  ]
}
  "modules": [
    {
      "name": "find-webcams",
      "module_id": "rand:find-webcams",
      "type": "registry"
    }
  ]
}


