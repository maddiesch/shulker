package shulker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"go.uber.org/zap"
)

type Params struct {
	ConfigFile string
}

type Config struct {
	WorkingDir string `hcl:"working_dir"`

	Java struct {
		Command string   `hcl:"command"`
		Flags   []string `hcl:"flags"`
	} `hcl:"java,block"`

	Minecraft struct {
		AutoRestart bool `hcl:"auto_restart"`

		Server struct {
			DownloadURL string `hcl:"download_url"`
			JarFile     string `hcl:"jar_file"`
		} `hcl:"server,block"`
	} `hcl:"minecraft,block"`
}

func (c Config) JavaCommand() (string, error) {
	if filepath.IsAbs(c.Java.Command) {
		return c.Java.Command, nil
	}
	return exec.LookPath(c.Java.Command)
}

func (c Config) ServerJar() string {
	if filepath.IsAbs(c.Minecraft.Server.JarFile) {
		return c.Minecraft.Server.JarFile
	}
	return filepath.Join(c.WorkingDir, c.Minecraft.Server.JarFile)
}

func NewConfig(p Params, log *zap.Logger) (Config, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	eval := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"os": cty.MapVal(map[string]cty.Value{
				"pwd": cty.StringVal(workingDir),
			}),
		},
		Functions: map[string]function.Function{
			`purpur_latest`: function.New(&function.Spec{
				Impl: hclFuncPurpurLatestImpl,
				Type: function.StaticReturnType(cty.String),
				Params: []function.Parameter{
					{Type: cty.String},
				},
			}),
		},
	}

	log.Debug("Loading Configuration", zap.String("path", p.ConfigFile))

	var config Config

	if err := hclsimple.DecodeFile(p.ConfigFile, eval, &config); err != nil {
		log.Error("Failed to decode configuration file", zap.Error(err))
		return Config{}, err
	}

	return config, nil
}

var hclFuncPurpurLatestImpl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.pl3x.net/v2/purpur/%s/", args[0].AsString()))
	if err != nil {
		return cty.NilVal, err
	}
	if resp.StatusCode != 200 {
		return cty.NilVal, fmt.Errorf("http error fetching Purpur latest %s", http.StatusText(resp.StatusCode))
	}

	builds := struct {
		Builds struct {
			Latest string `json:"latest"`
		} `json:"builds"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return cty.NilVal, err
	}

	return cty.StringVal(
		fmt.Sprintf("https://api.pl3x.net/v2/purpur/%s/%s/download", args[0].AsString(), builds.Builds.Latest),
	), nil
}
