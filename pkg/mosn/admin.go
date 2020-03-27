//  Copyright 2018 Istio Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package mosn

import (
	"bytes"
	"fmt"
	"istio.io/istio/pkg/proxy"
	"strings"

	envoyAdmin "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// Shutdown initiates a graceful shutdown of Envoy.
func Shutdown(adminPort uint32) error {
	_, err := doMosnPost("quitquitquit", "", "", adminPort)
	return err
}

// GetServerInfo returns a structure representing a call to /server_info
func GetServerInfo(adminPort uint32) (*envoyAdmin.ServerInfo, error) {
	buffer, err := doMosnGet("server_info", adminPort)
	if err != nil {
		return nil, err
	}

	msg := &envoyAdmin.ServerInfo{}
	if err := unmarshal(buffer.String(), msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// GetConfigDump polls Envoy admin port for the config dump and returns the response.
func GetConfigDump(adminPort uint32) (*envoyAdmin.ConfigDump, error) {
	buffer, err := doMosnGet("config_dump", adminPort)
	if err != nil {
		return nil, err
	}

	msg := &envoyAdmin.ConfigDump{}
	if err := unmarshal(buffer.String(), msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func doMosnGet(path string, adminPort uint32) (*bytes.Buffer, error) {
	requestURL := fmt.Sprintf("http://127.0.0.1:%d/%s", adminPort, path)
	buffer, err := proxy.DoHTTPGet(requestURL)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func doMosnPost(path, contentType, body string, adminPort uint32) (*bytes.Buffer, error) {
	requestURL := fmt.Sprintf("http://127.0.0.1:%d/%s", adminPort, path)
	buffer, err := proxy.DoHTTPPost(requestURL, contentType, body)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func unmarshal(jsonString string, msg proto.Message) error {
	u := jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
	return u.Unmarshal(strings.NewReader(jsonString), msg)
}
