job "sample" {
  type = "batch"
  datacenters = ["dc1"]

  group "runstuff" {
    task "dothething" {
      driver = "exec"
      config {
        command = "bash"
        args = ["-c", "echo hi"]
      }

      resources {
        device "doyota/car/mius" {}
      }
    }
  }
}

