job "another" {
  type = "batch"
  datacenters = ["dc1"]

  group "runstuff" {
    task "dothething" {
      driver = "raw_exec"
      config {
        command = "bash"
        args = ["-c", "echo hi && sleep 30"]
      }

      resources {
        device "doyota/car/mius" {}
      }
    }
  }
}

