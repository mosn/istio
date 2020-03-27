package mosn

import (
	"fmt"
	"io"
	"istio.io/istio/pkg/bootstrap"
)

const (
	// DefaultConfigFile is a defualt config file name for the root config JSON
	DefaultConfigFile = "config.json"
)

type mosnBootstrap struct {
	bootstrap.EnvoyBootstrap
}

func (m *mosnBootstrap) WriteTo(w io.Writer) error {
	return fmt.Errorf("this method should not be called normally")
}

func (m *mosnBootstrap) CreateFileForEpoch(epoch int) (string, error) {
	return m.EnvoyBootstrap.CreateFileForEpoch(epoch)
}

func newBootstrap(config bootstrap.Config) bootstrap.Instance {
	return &mosnBootstrap{
		bootstrap.EnvoyBootstrap{config},
	}
}
