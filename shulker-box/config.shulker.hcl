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
