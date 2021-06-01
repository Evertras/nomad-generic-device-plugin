job "sample" {
  type = "batch"
  datacenters = ["dc1"]

  group "runstuff" {
    task "dothething" {
      driver = "exec"
      config {
        command = "bash"
        args = ["-c", "echo hi && sleep 30s && echo done"]
      }

      resources {
        # Resources go <vendor>/<type>/<model> when specifying all three
        device "store-brand/anotherbox/another-model" {}
      }
    }
  }
}

