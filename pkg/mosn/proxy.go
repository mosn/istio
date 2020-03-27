// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mosn

import (
	envoyAdmin "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	_ "io/ioutil"
	"istio.io/istio/pkg/bootstrap"
	"path"

	"istio.io/pkg/log"
	"os"
	"os/exec"

	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/proxy"
)

const (
	defaultBinaryPath = "/usr/local/bin/mosn"
	CmdStart          = "start"
	ArgConfig         = "--config"
	ArgServiceCluster = "--service-cluster"
	ArgServiceNode    = "--service-node"
)

func init() {
	proxy.RegisterProxyFactory(constants.IstioProxyMosnImplement, newMosn)
}

func newMosn(config proxy.ProxyConfig) (proxy.Proxy, error) {
	// check binary path exists
	binaryPath := config.Config.BinaryPath
	// use default binary path if configure binary path is empty
	if config.Config.BinaryPath != "" {
		log.Infof("binary path from flag is empty, try using %s as binary path", defaultBinaryPath)
		binaryPath = defaultBinaryPath
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, err
	}

	config.Config.BinaryPath = binaryPath

	return &mosn{
		ProxyConfig: config,
	}, nil
}

type mosn struct {
	proxy.ProxyConfig
	//extraArgs []string
}

func (e *mosn) IsLive() bool {
	adminPort := uint32(e.Config.ProxyAdminPort)
	info, err := GetServerInfo(adminPort)
	if err != nil {
		log.Infof("failed retrieving server from Envoy on port %d: %v", adminPort, err)
		return false
	}

	// TODO 适配mosn的live接口
	if info.State == envoyAdmin.ServerInfo_LIVE {
		// It's live.
		return true
	}

	log.Infof("envoy server not yet live, state: %s", info.State.String())
	return false
}

func (e *mosn) Run(config interface{}, epoch int, abort <-chan error) error {
	var fname string
	// Note: the cert checking still works, the generated file is updated if certs are changed.
	// We just don't save the generated file, but use a custom one instead. Pilot will keep
	// monitoring the certs and restart if the content of the certs changes.

	// TODO 看 mosn 是否有必要
	//if _, ok := config.(proxy.DrainConfig); ok {
	//	// We are doing a graceful termination, apply an empty config to drain all connections
	//	//fname = drainFile
	//}

	if len(e.Config.CustomConfigFile) > 0 {
		// there is a custom configuration. Don't write our own config - but keep watching the certs.
		fname = e.Config.CustomConfigFile
	} else {
		out, err := newBootstrap(bootstrap.Config{
			Node:                e.Node,
			DNSRefreshRate:      e.DNSRefreshRate,
			Proxy:               &e.Config,
			PilotSubjectAltName: e.PilotSubjectAltName,
			MixerSubjectAltName: e.MixerSubjectAltName,
			LocalEnv:            os.Environ(),
			NodeIPs:             e.NodeIPs,
			PodName:             e.PodName,
			PodNamespace:        e.PodNamespace,
			PodIP:               e.PodIP,
			SDSUDSPath:          e.SDSUDSPath,
			SDSTokenPath:        e.SDSTokenPath,
			ControlPlaneAuth:    e.ControlPlaneAuth,
			DisableReportCalls:  e.DisableReportCalls,
		}).CreateFileForEpoch(epoch)
		if err != nil {
			log.Errora("Failed to generate bootstrap config: ", err)
			os.Exit(1) // Prevent infinite loop attempting to write the file, let k8s/systemd report
			return err
		}
		fname = out
	}

	// spin up a new Envoy process
	args := e.args(fname, epoch)
	log.Infof("Envoy command: %v", args)

	/* #nosec */
	cmd := exec.Command(e.Config.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-abort:
		log.Warnf("Aborting epoch %d", epoch)
		if errKill := cmd.Process.Kill(); errKill != nil {
			log.Warnf("killing epoch %d caused an error %v", epoch, errKill)
		}
		return err
	case err := <-done:
		return err
	}
}

func (e *mosn) Cleanup(epoch int) {
	filePath := path.Join(e.Config.ConfigPath, DefaultConfigFile)
	if err := os.Remove(filePath); err != nil {
		log.Warnf("Failed to delete config file %s for %d, %v", filePath, epoch, err)
	}
}

func (e *mosn) args(fname string, _ int) []string {
	startupArgs := []string{CmdStart,
		ArgConfig, fname,
		ArgServiceCluster, e.Config.ServiceCluster,
		ArgServiceNode, e.Node,
	}

	return startupArgs
}
