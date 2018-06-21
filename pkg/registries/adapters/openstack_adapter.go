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

package adapters

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/automationbroker/bundle-lib/apb"
	"github.com/automationbroker/bundle-lib/registries/adapters"
	log "github.com/sirupsen/logrus"
)

// OpenstackAdapter - Docker Hub Adapter
type OpenstackAdapter struct {
	Config adapters.Configuration
}

// RegistryName - Retrieve the registry name
func (r OpenstackAdapter) RegistryName() string {
	if r.Config.URL.Host == "" {
		return r.Config.URL.Path
	}
	return r.Config.URL.Host
}

// GetImageNames - retrieve the images
func (r OpenstackAdapter) GetImageNames() ([]string, error) {
	var apbNames []string
	var projects []string
	services := []string{"VM"}

	if len(r.Config.Org) == 0 {
		token, err := r.getUnscopedToken()
		if err != nil {
			return apbNames, err
		}
		projects, err = r.getProjects(token)
		if err != nil {
			return apbNames, err
		}
	} else {
		projects = append(projects, r.Config.Org)
	}

	for _, project := range projects {
		for _, service := range services {
			apbNames = append(apbNames, fmt.Sprintf("openstack-%v-%v-project-apb", service, project))
		}
	}

	return apbNames, nil
}

// FetchSpecs - retrieve the spec for the image names.
func (r OpenstackAdapter) FetchSpecs(imageNames []string) ([]*apb.Spec, error) {
	specs := []*apb.Spec{}
	log.Warningf("Entered FetchSpecs, %v", imageNames)

	for _, imageName := range imageNames {
		spec, err := r.loadSpec(imageName)
		if err != nil {
			log.Errorf("Failed to retrieve spec data for image %s - %v", imageName, err)
		}
		if spec != nil {
			specs = append(specs, spec)
		}
	}
	log.Warningf("Leaving FetchSpecs, %v", specs)
	return specs, nil
}

func (r OpenstackAdapter) loadSpec(imageName string) (*apb.Spec, error) {
	log.Warningf("entered OpenstackAdapter.loadSpec(%s)", imageName)
	var spec apb.Spec
	var plan apb.Plan
	var parameters []apb.ParameterDescriptor
	splitName := strings.Split(imageName, "-")
	splitlen := len(splitName)
	service := splitName[1]
	project := strings.Join(splitName[2:(splitlen-2)], "-")
	displayName := fmt.Sprintf("Openstack %v in %v Project (APB)", service, project)

	//Configure Plan
	plan.Name = "default"
	plan.Parameters = parameters

	//Configure APB
	spec.Runtime = 2
	spec.Description = fmt.Sprintf("Provisions an Openstack %v instance in the %v Project using a Heat Template", service, project)
	spec.Image = r.Config.Runner
	spec.FQName = strings.Replace(imageName, "_", "-", -1)
	spec.Version = "1.0"
	spec.Bindable = false
	spec.Async = "optional"
	spec.Metadata = map[string]interface{}{
		"displayName":         displayName,
		"providerDisplayName": "Red Hat, Inc.",
	}
	spec.Plans = append(spec.Plans, plan)

	log.Warningf("leaving OpenstackAdapter.loadSpec(%s), returning %v", imageName, spec)
	return &spec, nil
}

func (r OpenstackAdapter) getUnscopedToken() (string, error) {
	authString := fmt.Sprintf("{ \"auth\": { \"identity\": { \"methods\": [\"password\"], \"password\": { \"user\": { \"name\": \"%v\", \"domain\": { \"id\": \"default\" }, \"password\": \"%v\" }}}}}", r.Config.User, r.Config.Pass)
	authBytes := []byte(authString)

	authUrl := fmt.Sprintf("%v/identity/v3/auth/tokens",
		r.Config.URL.String())

	response, err := openstackRequest(authUrl, "POST", authBytes, "")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	return response.Header["X-Subject-Token"][0], nil
}

func (r OpenstackAdapter) getProjects(token string) ([]string, error) {
	var projects []string

	type Project struct {
		Name string `json:"name"`
	}

	type ProjectResponse struct {
		Projects []Project `json:"projects"`
	}

	projectUrl := fmt.Sprintf("%v/identity/v3/auth/projects",
		r.Config.URL.String())
	response, err := openstackRequest(projectUrl, "GET", nil, token)
	if err != nil {
		return []string{}, err
	}
	defer response.Body.Close()
	projectJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []string{}, err
	}

	projectResponse := ProjectResponse{}
	err = json.Unmarshal(projectJson, &projectResponse)
	if err != nil {
		return []string{}, err
	}

	for _, project := range projectResponse.Projects {
		projects = append(projects, project.Name)
	}

	return projects, nil
}

func openstackRequest(requestUrl string, method string, data []byte, token string) (*http.Response, error) {
	req, err := http.NewRequest(method, requestUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(token) != 0 {
		req.Header.Set("X-Auth-Token", token)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: transport}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, errors.New(resp.Status)
	}
	response := resp

	log.Warningf("Request completed successfully")
	return response, nil
}
