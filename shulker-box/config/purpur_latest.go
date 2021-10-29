package config

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zclconf/go-cty/cty"
)

var purpurLatestFuncHandler = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
	Log.Debugf(`Resolve Purpur latest for Minecraft version %s`, args[0].AsString())

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
