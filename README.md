# Shulker

A Minecraft Container & Controller

## Usage

_warning:_ This software is in very early development. It can be used to run a Minecraft server, but I wouldn't reccomend it for production usecases.

### Configuration

Shulker uses [HCL](https://github.com/hashicorp/hcl) for configuration.

Example config.

```hcl
working_dir = os.pwd

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
```

The provided `purpur_latest` function will query the latest version of the [Purpur](https://purpurmc.org) Minecraft server for the specified Minecraft version.
