//
// Copyright (c) 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"fmt"
	"os"

	"github.com/automationbroker/bundle-lib/registries"
	"github.com/automationbroker/config"
	flags "github.com/jessevdk/go-flags"
	"github.com/openshift/ansible-service-broker/pkg/app"
	"github.com/openshift/ansible-service-broker/pkg/version"
	"github.com/openstack/openstack-broker/pkg/registries/adapters"
	log "github.com/sirupsen/logrus"
)

func main() {

	var args app.Args
	var err error

	// To add your custom registries, define an entry in this array.
	regs := []registries.Registry{}
	oadapter := adapters.OpenstackAdapter{Name: "openstack"}

	brokerconfig, err := config.CreateConfig("/etc/openstackbroker/config.yaml")
	if err != nil {
		os.Stderr.WriteString("ERROR: Failed to read config file\n")
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	for _, config := range brokerconfig.GetSubConfigArray("registry") {
		c := registries.Config{
			URL:        config.GetString("url"),
			User:       config.GetString("user"),
			Pass:       config.GetString("pass"),
			Org:        config.GetString("org"),
			Tag:        config.GetString("tag"),
			Type:       config.GetString("type"),
			Name:       config.GetString("name"),
			Images:     config.GetSliceOfStrings("images"),
			Namespaces: config.GetSliceOfStrings("namespaces"),
			Fail:       config.GetBool("fail_on_error"),
			WhiteList:  config.GetSliceOfStrings("white_list"),
			BlackList:  config.GetSliceOfStrings("black_list"),
			AuthType:   config.GetString("auth_type"),
			AuthName:   config.GetString("auth_name"),
			Runner:     config.GetString("runner"),
		}
		reg, err := registries.NewCustomRegistry(c, oadapter, "openstack")
		if err != nil {
			log.Errorf(
				"Failed to initialize %v Registry err - %v \n", config.GetString("name"), err)
			os.Exit(1)
		}
		regs = append(regs, reg)
	}

	// Writing directly to stderr because log has not been bootstrapped
	if args, err = app.CreateArgs(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if args.Version {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	// CreateApp passing in the args and registries
	app := app.CreateApp(args, regs)
	app.Start()
}
