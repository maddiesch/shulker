working_dir = "${os.pwd}/shulker/data"

java {
  command = "java"
  flags = []
}

minecraft {
  auto_restart = true

  server {
    download_url = purpur_latest("1.18.2")
    jar_file = "server.jar"
  }
}

controller "unix" {
  listen_on = "/tmp/shulker.sock"

  identity "shulker" {
    password = env("SHULKER_PASSWORD")
    access_level = "ADMIN"
  }
}
