package utils

import (
	"github.com/BurntSushi/toml"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"strings"
)

// Execute - This code was copied from frpc and modified a bit to support requirements
func ParseClientConfig(toml []byte) (
	cliCfg *v1.ClientCommonConfig,
	pxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	err error,
) {
	allCfg := &v1.ClientConfig{}
	err = config.LoadConfigure(toml, &allCfg)
	if err != nil {
		return
	}
	cliCfg = &allCfg.ClientCommonConfig

	for _, c := range allCfg.Proxies {
		pxyCfgs = append(pxyCfgs, c.ProxyConfigurer)
	}
	for _, c := range allCfg.Visitors {
		visitorCfgs = append(visitorCfgs, c.VisitorConfigurer)
	}
	if cliCfg != nil {
		cliCfg.Complete()
	}
	for _, c := range pxyCfgs {
		c.Complete(cliCfg.User)
	}
	for _, c := range visitorCfgs {
		c.Complete(cliCfg)
	}
	return cliCfg, pxyCfgs, visitorCfgs, nil
}

func Marshal(v interface{}) (string, error) {

	sb := strings.Builder{}

	err := toml.NewEncoder(&sb).Encode(&v)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}
