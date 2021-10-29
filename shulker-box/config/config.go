package config

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"shulker-box/logger"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var Log = logger.L.WithField(`subsystem`, `configuration`)

type Config struct {
	WorkingDir string `hcl:"working_dir"`
	Minecraft  struct {
		AutoRestart bool `hcl:"auto_restart"`
		Java        struct {
			Command string   `hcl:"command"`
			Flags   []string `hcl:"flags"`
		} `hcl:"java,block"`
		Server struct {
			DownloadURL string `hcl:"download_url"`
			JarPath     string `hcl:"jar_file"`
		} `hcl:"server,block"`
		Plugins []MinecraftPlugin `hcl:"plugin,block"`
	} `hcl:"minecraft,block"`
	ControlServer struct {
		Port  int                 `hcl:"port"`
		Host  string              `hcl:"host"`
		Users []ControlServerUser `hcl:"user,block"`
	} `hcl:"control_server,block"`
}

type ControlServerUser struct {
	Username string `hcl:"name,label"`
	Password string `hcl:"password"`
}

type MinecraftPlugin struct {
	Name     string `hcl:"name,label"`
	Source   string `hcl:"source"`
	Required bool   `hcl:"required,optional"`
}

func Load(ctx context.Context, path string) (Config, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	Log.Infof(`Loading configuration from %s`, path)

	root := struct {
		Config Config `hcl:"shulker,block"`
	}{}

	exec := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			`pwd`: cty.StringVal(workingDir),
		},
		Functions: map[string]function.Function{
			`purpur_latest`: function.New(&function.Spec{
				Impl: purpurLatestFuncHandler,
				Type: function.StaticReturnType(cty.String),
				Params: []function.Parameter{
					{Type: cty.String},
				},
			}),
		},
	}

	if err := hclsimple.DecodeFile(path, exec, &root); err != nil {
		return Config{}, err
	}

	Log.Tracef(`Parsed Shulker Configuration (%s)`, path)

	return root.Config, nil
}

func (c Config) ServerJar() string {
	if filepath.IsAbs(c.Minecraft.Server.JarPath) {
		return c.Minecraft.Server.JarPath
	}
	return c.WorkingDirRelative(c.Minecraft.Server.JarPath)
}

func (c Config) WorkingDirRelative(p ...string) string {
	components := append([]string{c.WorkingDir}, p...)

	return filepath.Join(components...)
}

func (c Config) JavaCommand() (string, error) {
	if filepath.IsAbs(c.Minecraft.Java.Command) {
		return c.Minecraft.Java.Command, nil
	}
	return exec.LookPath(c.Minecraft.Java.Command)
}

func (c Config) ControlServerAddr() string {
	return net.JoinHostPort(c.ControlServer.Host, strconv.Itoa(c.ControlServer.Port))
}
