package main

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
)

type shulkerConfig struct {
	LogPath    string `hcl:"log_file"`
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
	} `hcl:"minecraft,block"`
}

func loadAndParseShulkerConfigAtFilePath(path string) (shulkerConfig, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return shulkerConfig{}, err
	}

	root := struct {
		Config shulkerConfig `hcl:"shulker,block"`
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
		return shulkerConfig{}, err
	}

	cfg := root.Config

	if err := finalizeShulkerConfig(&cfg); err != nil {
		return shulkerConfig{}, err
	}

	return cfg, nil
}

func finalizeShulkerConfig(cfg *shulkerConfig) error {
	if !filepath.IsAbs(cfg.LogPath) {
		cfg.LogPath = filepath.Join(cfg.WorkingDir, cfg.LogPath)
	}
	if !filepath.IsAbs(cfg.Minecraft.Server.JarPath) {
		cfg.Minecraft.Server.JarPath = filepath.Join(cfg.WorkingDir, cfg.Minecraft.Server.JarPath)
	}
	if !filepath.IsAbs(cfg.Minecraft.Java.Command) {
		full, err := exec.LookPath(cfg.Minecraft.Java.Command)
		if err != nil {
			return err
		}
		cfg.Minecraft.Java.Command = full
	}

	return nil
}

var purpurLatestFuncHandler = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.pl3x.net/v2/purpur/%s/", args[0].AsString()))
	if err != nil {
		return cty.NilVal, err
	}
	if resp.StatusCode != 200 {
		return cty.NilVal, fmt.Errorf("http error fetching purpur latest %s", http.StatusText(resp.StatusCode))
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
