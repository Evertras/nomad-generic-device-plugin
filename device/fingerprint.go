package device

import (
	"context"
	"time"

	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/plugins/device"
)

// doFingerprint is the long-running goroutine that detects device changes
func (d *GenericDevicePlugin) doFingerprint(ctx context.Context, devices chan *device.FingerprintResponse) {
	defer close(devices)

	// Create a timer that will fire immediately for the first detection
	ticker := time.NewTimer(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticker.Reset(d.fingerprintPeriod)
		}

		d.writeFingerprintToChannel(devices)
	}
}

// fingerprintedDevice is what we "discover" and transform into device.Device objects.
//
// plugin implementations will likely have a native struct provided by the corresonding SDK
type fingerprintedDevice struct {
	ID       string
	config   GenericDeviceConfig
	PCIBusID string
}

// writeFingerprintToChannel collects fingerprint info, partitions devices into
// device groups, and sends the data over the provided channel.
func (d *GenericDevicePlugin) writeFingerprintToChannel(devices chan<- *device.FingerprintResponse) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if len(d.identifiedDevices) == 0 {
		// "discover" the devices we have configured
		discoveredDevices := make([]*fingerprintedDevice, 0)

		for _, device := range d.configuredDevices {
			discoveredDevices = append(discoveredDevices, &fingerprintedDevice{
				// TODO: When is this okay to change?
				ID:     uuid.Generate(),
				config: device,
				// TODO: Do we need this?
				PCIBusID: uuid.Generate(),
			})
		}

		d.logger.Info("Found devices", "count", len(discoveredDevices))

		// during fingerprinting, devices are grouped by "device group" in
		// order to facilitate scheduling
		// devices in the same device group should have the same
		// Vendor, Type, and Name ("Model")
		// Build Fingerprint response with computed groups and send it over the channel
		deviceListByDeviceName := make(map[string][]*fingerprintedDevice)
		for _, device := range discoveredDevices {
			deviceName := device.config.Model
			deviceListByDeviceName[deviceName] = append(deviceListByDeviceName[deviceName], device)
			d.identifiedDevices[device.ID] = device.config
		}

		// Build Fingerprint response with computed groups and send it over the channel
		deviceGroups := make([]*device.DeviceGroup, 0, len(deviceListByDeviceName))
		for groupName, devices := range deviceListByDeviceName {
			deviceGroups = append(deviceGroups, deviceGroupFromFingerprintData(groupName, devices))
		}

		devices <- device.NewFingerprint(deviceGroups...)
	}
}

// deviceGroupFromFingerprintData composes deviceGroup from a slice of detected devices
func deviceGroupFromFingerprintData(groupName string, deviceList []*fingerprintedDevice) *device.DeviceGroup {
	// deviceGroup without devices makes no sense -> return nil when no devices are provided
	if len(deviceList) == 0 {
		return nil
	}

	devices := make([]*device.Device, 0, len(deviceList))
	for _, dev := range deviceList {
		devices = append(devices, &device.Device{
			ID:      dev.ID,
			Healthy: true,
			// TODO: Do we need this?
			HwLocality: &device.DeviceLocality{
				PciBusID: dev.PCIBusID,
			},
		})
	}

	deviceGroup := &device.DeviceGroup{
		// TODO: is this a valid assumption?
		Vendor: deviceList[0].config.Vendor,
		// TODO: is this a valid assumption?
		Type: deviceList[0].config.Type,

		Name:    groupName,
		Devices: devices,
		// The device API assumes that devices with the same DeviceName have the same
		// attributes like amount of memory, power, bar1memory, etc.
		// If not, then they'll need to be split into different device groups
		// with different names.
		/*
			Attributes: map[string]*structs.Attribute{
				"attrA": {
					Int:  helper.Int64ToPtr(1024),
					Unit: "MB",
				},
				"attrB": {
					Float: helper.Float64ToPtr(10.5),
					Unit:  "MW",
				},
			},
		*/
	}
	return deviceGroup
}
