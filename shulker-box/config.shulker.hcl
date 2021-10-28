shulker {
  log_file = "./logs/shulker.log"
  working_dir = "${pwd}/shulker/data"

  minecraft {
    auto_restart = true

    java {
      command = "java"
      flags = []
    }
    server {
      download_url = purpur_latest("1.17.1")
      jar_file = "./server.jar"
    }
  }

  control_server {
    port = 3000
    host = "0.0.0.0"

    user "admin" {
      password = "password"
    }
  }
}
