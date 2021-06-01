Nomad Generic Device Plugin
===========================

Nomad allows jobs to request various devices to reserve, such as GPUs.  However,
there is no way to specify generic devices to reserve, such as a mobile device
or external hardware.

This plugin uses the [Nomad device plugins](https://www.nomadproject.io/docs/internals/plugins/devices.html)
feature to allow you to configure arbitrary hardware devices to be available
on the node for reservation.

Based on [the Skeleton Device Plugin](https://github.com/hashicorp/nomad-skeleton-device-plugin)
provided by Hashicorp.

Features
--------

Currently, the plugin allows you to specify arbitrary values for a device.
No checks are made.  If you configure a device in the client config, the device
will be made available for reservation.

Only single, discrete devices are currently supported.

Wishlist
--------

Arbitrary health check scripts to see if the device is actually healthy/available.

Arbitrary attributes that can be selected against, such as memory usage.

Attribute/model scripts that can dynamically fill in data when fingerprinted.

Better tests and cleaner code, this was bare minimum changes from the skeleton
reference with some manual testing to see what actually works.

Requirements
------------

- [Go](https://golang.org/doc/install) 1.12 or later (to build the plugin)

Building the Generic Device Plugin
----------------------------------

```sh
$ make build
```

Running the Plugin in Development
---------------------------------

You can test this plugin (and your own device plugins) in development using the
[plugin launcher](https://github.com/hashicorp/nomad/tree/master/plugins/shared/cmd/launcher). The makefile provides
a target for this:

```sh
$ make eval
```

Deploying Device Plugins in Nomad
---------------------------------

Copy the plugin binary to the
[plugins directory](https://www.nomadproject.io/docs/configuration/index.html#plugin_dir) and
[configure the plugin](https://www.nomadproject.io/docs/configuration/plugin.html) in the client config. Then use the
[device stanza](https://www.nomadproject.io/docs/job-specification/device.html) in the job file to schedule with
device support.
