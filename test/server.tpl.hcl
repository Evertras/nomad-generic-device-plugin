################################################################################
# Local test Nomad config
#
# The .tpl file is intended as a starting point.  The Makefile will update things
# such as the data dir which require an absolute path which cannot be written
# in git.  To regenerate it, delete the generated file and run 'make nomad-test-server'

data_dir  = "PWD/test/nomad/data"
plugin_dir = "PWD/test/nomad/plugins"

bind_addr = "0.0.0.0" # the default

server {
  enabled          = true
  bootstrap_expect = 1
}

client {
  enabled = true
}

plugin "raw_exec" {
  config {
    enabled = true
  }
}

plugin "generic-device" {
  config {
    device {
      type = "phone"
      vendor = "woogle"
      model = "nebula10"
    }

    device {
      type = "phone"
      vendor = "mango"
      model = "mphone12-max"
    }

    device {
      type = "car"
      vendor = "doyota"
      model = "mius"
    }
  }
}
