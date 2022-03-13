package device

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/device"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/kr/pretty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// pluginName is the deviceName of the plugin
	// this is used for logging and (along with the version) for uniquely identifying
	// plugin binaries fingerprinted by the client
	pluginName = "generic-device-plugin"

	// plugin version allows the client to identify and use newer versions of
	// an installed plugin
	pluginVersion = "v0.1.0"
)

var (
	// pluginInfo provides information used by Nomad to identify the plugin
	pluginInfo = &base.PluginInfoResponse{
		Type:              base.PluginTypeDevice,
		PluginApiVersions: []string{device.ApiVersion010},
		PluginVersion:     pluginVersion,
		Name:              pluginName,
	}

	// configSpec is the specification of the schema for this plugin's config.
	// this is used to validate the HCL for the plugin provided
	// as part of the client config:
	//   https://www.nomadproject.io/docs/configuration/plugin.html
	// options are here:
	//   https://github.com/hashicorp/nomad/blob/v0.10.0/plugins/shared/hclspec/hcl_spec.proto
	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"device": hclspec.NewBlockList("device", hclspec.NewObject(map[string]*hclspec.Spec{
			"vendor": hclspec.NewAttr("vendor", "string", true),
			"model":  hclspec.NewAttr("model", "string", true),
			"type":   hclspec.NewAttr("type", "string", true),
		})),
		"fingerprint_period": hclspec.NewDefault(
			hclspec.NewAttr("fingerprint_period", "string", false),
			hclspec.NewLiteral("\"1m\""),
		),
	})
)

type GenericDeviceConfig struct {
	Type   string `codec:"type"`
	Vendor string `codec:"vendor"`
	Model  string `codec:"model"`
}

// Config contains configuration information for the plugin.
type Config struct {
	FingerprintPeriod string `codec:"fingerprint_period"`

	Devices []GenericDeviceConfig `codec:"device"`
}

// GenericDevicePlugin claims to have arbitrary devices as defined by config.hcl
// and optionally checks for their availability with the provided script
type GenericDevicePlugin struct {
	logger log.Logger

	// these are local copies of the config values that we need for operation
	someString   string
	someBool     bool
	someIntArray []int

	// a list of devices that we claim to have
	configuredDevices []GenericDeviceConfig

	// devices mapped with generated IDs for reservation purposes
	identifiedDevices map[string]GenericDeviceConfig
	deviceLock        sync.RWMutex

	// fingerprintPeriod the period for the fingerprinting loop
	// most plugins that fingerprint in a polling loop will have this
	fingerprintPeriod time.Duration
}

// NewPlugin returns a device plugin, used primarily by the main wrapper
//
// Plugin configuration isn't available yet, so there will typically be
// a limit to the initialization that can be performed at this point.
func NewPlugin(log log.Logger) *GenericDevicePlugin {
	return &GenericDevicePlugin{
		logger:            log.Named(pluginName),
		configuredDevices: nil,
		identifiedDevices: make(map[string]GenericDeviceConfig),
	}
}

// PluginInfo returns information describing the plugin.
//
// This is called during Nomad client startup, while discovering and loading
// plugins.
func (d *GenericDevicePlugin) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the configuration schema for the plugin.
//
// This is called during Nomad client startup, immediately before parsing
// plugin config and calling SetConfig
func (d *GenericDevicePlugin) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is called by the client to pass the configuration for the plugin.
func (d *GenericDevicePlugin) SetConfig(c *base.Config) error {

	// decode the plugin config
	var config Config
	if err := base.MsgPackDecode(c.PluginConfig, &config); err != nil {
		return err
	}

	period, err := time.ParseDuration(config.FingerprintPeriod)
	if err != nil {
		return fmt.Errorf("failed to parse doFingerprint period %q: %v", config.FingerprintPeriod, err)
	}
	d.fingerprintPeriod = period
	d.configuredDevices = config.Devices

	d.logger.Debug("test debug")
	d.logger.Info("config set", "config", log.Fmt("% #v", pretty.Formatter(config)))
	return nil
}

// Fingerprint streams detected devices.
// Messages should be emitted to the returned channel when there are changes
// to the devices or their health.
func (d *GenericDevicePlugin) Fingerprint(ctx context.Context) (<-chan *device.FingerprintResponse, error) {
	// Fingerprint returns a channel. The recommended way of organizing a plugin
	// is to pass that into a long-running goroutine and return the channel immediately.
	outCh := make(chan *device.FingerprintResponse)
	go d.doFingerprint(ctx, outCh)
	return outCh, nil
}

// Stats streams statistics for the detected devices.
// Messages should be emitted to the returned channel on the specified interval.
func (d *GenericDevicePlugin) Stats(ctx context.Context, interval time.Duration) (<-chan *device.StatsResponse, error) {
	// Similar to Fingerprint, Stats returns a channel. The recommended way of
	// organizing a plugin is to pass that into a long-running goroutine and
	// return the channel immediately.
	outCh := make(chan *device.StatsResponse)
	go d.doStats(ctx, outCh, interval)
	return outCh, nil
}

type reservationError struct {
	notExistingIDs []string
}

func (e *reservationError) Error() string {
	return fmt.Sprintf("unknown device IDs: %s", strings.Join(e.notExistingIDs, ","))
}

// Reserve returns information to the task driver on on how to mount the given devices.
// It may also perform any device-specific orchestration necessary to prepare the device
// for use. This is called in a pre-start hook on the client, before starting the workload.
func (d *GenericDevicePlugin) Reserve(deviceIDs []string) (*device.ContainerReservation, error) {
	if len(deviceIDs) == 0 {
		return &device.ContainerReservation{}, nil
	}

	d.logger.Debug("Asked to reserve", "ids", deviceIDs)

	// This pattern can be useful for some drivers to avoid a race condition where a device disappears
	// after being scheduled by the server but before the server gets an update on the fingerprint
	// channel that the device is no longer available.
	d.deviceLock.RLock()
	var notExistingIDs []string
	for _, id := range deviceIDs {
		if _, deviceIDExists := d.identifiedDevices[id]; !deviceIDExists {
			notExistingIDs = append(notExistingIDs, id)
		}
	}
	d.deviceLock.RUnlock()
	if len(notExistingIDs) != 0 {
		return nil, &reservationError{notExistingIDs}
	}

	// initialize the response
	resp := &device.ContainerReservation{
		Envs:    map[string]string{},
		Mounts:  []*device.Mount{},
		Devices: []*device.DeviceSpec{},
	}

	// Mounts are used to mount host volumes into a container that may include
	// libraries, etc.
	/*
		DISABLED for now as not surewhat needs mounting...
		resp.Mounts = append(resp.Mounts, &device.Mount{
			TaskPath: "/usr/lib/libsome-library.so",
			HostPath: "/usr/lib/libprobably-some-fingerprinted-or-configured-library.so",
			ReadOnly: true,
		})
	*/

	for i, id := range deviceIDs {
		// Check if the device is known
		if _, ok := d.identifiedDevices[id]; !ok {
			return nil, status.Newf(codes.InvalidArgument, "unknown device %q", id).Err()
		}

		// Envs are a set of environment variables to set for the task.
		resp.Envs[fmt.Sprintf("DEVICE_%d", i)] = id

		/*
			// DISABLED for now since the devices aren't mounted, leaving for reference
			// Devices are the set of devices to mount into the container.
			resp.Devices = append(resp.Devices, &device.DeviceSpec{
				// TaskPath is the location to mount the device in the task's file system.
				TaskPath: fmt.Sprintf("/dev/skel%d", i),
				// HostPath is the host location of the device.
				HostPath: fmt.Sprintf("/dev/devActual"),
				// CgroupPerms defines the permissions to use when mounting the device.
				CgroupPerms: "rx",
			})
		*/
	}

	return resp, nil
}
