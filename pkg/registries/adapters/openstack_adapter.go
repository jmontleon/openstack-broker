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
	"strconv"
	"strings"

	"github.com/automationbroker/bundle-lib/apb"
	"github.com/automationbroker/bundle-lib/registries/adapters"
	log "github.com/sirupsen/logrus"
)

// OpenstackAdapter - Docker Hub Adapter
type OpenstackAdapter struct {
	Config adapters.Configuration
}

type Object struct {
	Name      string `json:"name"`
	ProjectId string `json:"project_id,omitempty"`
}

type Project struct {
	ID string `json:"id"`
}

type Token struct {
	Project Project `json:"project"`
}

type TokenResponse struct {
	Token Token `json:"token"`
}

const unscopedAuthString = "{ \"auth\": { \"identity\": { \"methods\": [\"password\"], \"password\": { \"user\": { \"name\": \"%v\", \"domain\": { \"id\": \"default\" }, \"password\": \"%v\" }}}}}"
const scopedAuthString = "{ \"auth\": { \"identity\": { \"methods\": [\"password\"], \"password\": { \"user\": { \"name\": \"%v\", \"domain\": { \"id\": \"default\" }, \"password\": \"%v\" }}}, \"scope\": { \"project\": { \"name\": \"%v\",\"domain\": { \"id\": \"default\" }}}}}"

var services = []string{"vm"}

var parameterTypes = map[string][]map[string]string{
	"vm": {
		{"name": "flavors", "label": "Flavor", "path": "/compute/v2/flavors", "required": "true"},
		{"name": "keys", "label": "Key", "path": "/compute/v2/os-keypairs", "required": "false"},
		{"name": "images", "label": "Image", "path": "/compute/v2/images", "required": "true"},
		{"name": "networks", "label": "Network", "path": ":9696/v2.0/networks", "required": "true"},
		{"name": "security_groups", "label": "Security Group", "path": "/compute/v2/os-security-groups", "required": "false"},
	},
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

	if len(r.Config.Org) == 0 {
		token, err := r.getUnscopedToken()
		if err != nil {
			return apbNames, err
		}
		projects, err = r.getObjectList(token, "projects", "/identity/v3/auth/projects", "")
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
	log.Warningf("entered OpenstackAdapter.loadSpec(%v)", imageName)
	var spec apb.Spec
	var plan apb.Plan
	var parameters []apb.ParameterDescriptor
	splitName := strings.Split(imageName, "-")
	splitlen := len(splitName)
	service := splitName[1]
	project := strings.Join(splitName[2:(splitlen-2)], "-")
	displayName := fmt.Sprintf("Openstack %v in %v project (APB)", service, project)

	token, projectId, err := r.getScopedToken(project)
	if err != nil {
		log.Warningf("Could not get a scoped token: %s", err)
	}

	//Configure Parameters
	for _, pt := range parameterTypes[service] {
		values, err := r.getObjectList(token, pt["name"], pt["path"], projectId)
		if err != nil {
			log.Warningf("Could not retrieve %s: %s", pt["name"], err)
		}
		required, err := strconv.ParseBool(pt["required"])
		if err != nil {
			required = false
		}

		parameter := apb.ParameterDescriptor{
			Name:      strings.Replace(strings.ToLower(pt["label"]), " ", "_", -1),
			Title:     pt["label"],
			Type:      "enum",
			Updatable: false,
			Enum:      values,
			Required:  required,
		}
		if len(values) > 0 {
			parameter.Default = values[0]
		}
		parameters = append(parameters, parameter)

	}

	authParameters := [5]map[string]string{
		{"name": "url", "title": "URL", "default": fmt.Sprintf("%v/identity", r.Config.URL.String()), "type": "string", "displaytype": ""},
		{"name": "user", "title": "User", "default": r.Config.User, "type": "string", "displaytype": ""},
		{"name": "pass", "title": "Password", "default": r.Config.Pass, "type": "string", "displaytype": "password"},
		{"name": "project", "title": "Project", "default": project, "type": "string", "displaytype": ""},
		{"name": "service", "title": "Service", "default": service, "type": "string", "displaytype": ""},
	}

	for _, authParameter := range authParameters {
		parameter := apb.ParameterDescriptor{
			Name:         authParameter["name"],
			Title:        authParameter["title"],
			Type:         authParameter["type"],
			Updatable:    false,
			Required:     true,
			Default:      authParameter["default"],
			DisplayType:  authParameter["displaytype"],
			DisplayGroup: "Openstack Authentication",
		}
		parameters = append(parameters, parameter)
	}

	//Configure Plan
	plan.Name = "default"
	plan.Parameters = parameters
	plan.Description = fmt.Sprintf("Provisions an Openstack %v instance in the %v Project using a Heat Template", service, project)

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
	authString := fmt.Sprintf(unscopedAuthString, r.Config.User, r.Config.Pass)
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

func (r OpenstackAdapter) getScopedToken(project string) (string, string, error) {
	authString := fmt.Sprintf(scopedAuthString, r.Config.User, r.Config.Pass, project)
	authBytes := []byte(authString)

	authUrl := fmt.Sprintf("%v/identity/v3/auth/tokens",
		r.Config.URL.String())

	response, err := openstackRequest(authUrl, "POST", authBytes, "")
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	objectJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", "", err
	}

	objectResponse := TokenResponse{}
	err = json.Unmarshal(objectJson, &objectResponse)
	if err != nil {
		return "", "", err
	}

	return response.Header["X-Subject-Token"][0], objectResponse.Token.Project.ID, nil
}

func (r OpenstackAdapter) getObjectList(token string, objectType string, objectPath string, projectId string) ([]string, error) {
	var objects []string

	objectUrl := fmt.Sprintf("%v%v", r.Config.URL.String(), objectPath)
	if objectType == "networks" {
		objectUrl = strings.Replace(objectUrl, "https://", "http://", 1)
	}

	response, err := openstackRequest(objectUrl, "GET", nil, token)
	if err != nil {
		return []string{}, err
	}
	defer response.Body.Close()
	objectJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []string{}, err
	}

	var objectArray []Object
	switch objectType {
	case "keys":
		objectResponse := make(map[string][]map[string]Object)
		json.Unmarshal(objectJson, &objectResponse)
		if len(objectResponse["keypairs"]) == 0 {
			log.Warningf("Did not find any %v when unmarshalling response", objectType)
		}
		var objectList []Object
		for _, object := range objectResponse["keypairs"] {
			objectList = append(objectList, object["keypair"])
		}
		objectArray = objectList
	case "networks":
		objectResponse := make(map[string][]Object)
		json.Unmarshal(objectJson, &objectResponse)
		if len(objectResponse[objectType]) == 0 {
			log.Warningf("Did not find any %v when unmarshalling response", objectType)
		}
		n := 0
		for _, object := range objectResponse[objectType] {
			if object.ProjectId == projectId {
				objectResponse[objectType][n] = object
				n++
			}
		}
		objectResponse[objectType] = objectResponse[objectType][:n]
		objectArray = objectResponse[objectType]
	default:
		objectResponse := make(map[string][]Object)
		json.Unmarshal(objectJson, &objectResponse)
		if len(objectResponse[objectType]) == 0 {
			log.Warningf("Did not find any %v when unmarshalling response", objectType)
		}
		objectArray = objectResponse[objectType]
	}

	for _, object := range objectArray {
		objects = append(objects, object.Name)
	}

	return objects, nil
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
