shulker {
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

    plugin "ChestSort" {
      source = "https://www.spigotmc.org/resources/chestsort-api.59773/download?version=424811"
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
