package cfclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type ServiceKeysResponse struct {
	Count     int                  `json:"total_results"`
	Pages     int                  `json:"total_pages"`
	Resources []ServiceKeyResource `json:"resources"`
}

type ServiceKeyResource struct {
	Meta   Meta       `json:"metadata"`
	Entity ServiceKey `json:"entity"`
}

type CreateServiceKeyRequest struct {
	Name                string      `json:"name"`
	ServiceInstanceGuid string      `json:"service_instance_guid"`
	Parameters          interface{} `json:"parameters,omitempty"`
}

type ServiceKey struct {
	Name                string      `json:"name"`
	Guid                string      `json:"guid"`
	ServiceInstanceGuid string      `json:"service_instance_guid"`
	Credentials         interface{} `json:"credentials"`
	ServiceInstanceUrl  string      `json:"service_instance_url"`
	c                   *Client
}

func (c *Client) ListServiceKeysByQuery(query url.Values) ([]ServiceKey, error) {
	var serviceKeys []ServiceKey
	var serviceKeysResp ServiceKeysResponse
	r := c.NewRequest("GET", "/v2/service_keys?"+query.Encode())
	resp, err := c.DoRequest(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error requesting service keys")
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading service keys request:")
	}

	err = json.Unmarshal(resBody, &serviceKeysResp)
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling service keys")
	}
	for _, serviceKey := range serviceKeysResp.Resources {
		serviceKey.Entity.Guid = serviceKey.Meta.Guid
		serviceKey.Entity.c = c
		serviceKeys = append(serviceKeys, serviceKey.Entity)
	}
	return serviceKeys, nil
}

func (c *Client) ListServiceKeys() ([]ServiceKey, error) {
	return c.ListServiceKeysByQuery(nil)
}

func (c *Client) GetServiceKeyByName(name string) (ServiceKey, error) {
	var serviceKey ServiceKey
	q := url.Values{}
	q.Set("q", "name:"+name)
	serviceKeys, err := c.ListServiceKeysByQuery(q)
	if err != nil {
		return serviceKey, err
	}
	if len(serviceKeys) == 0 {
		return serviceKey, fmt.Errorf("Unable to find service key %s", name)
	}
	return serviceKeys[0], nil
}

// GetServiceKeyByInstanceGuid is deprecated in favor of GetServiceKeysByInstanceGuid
func (c *Client) GetServiceKeyByInstanceGuid(guid string) (ServiceKey, error) {
	var serviceKey ServiceKey
	q := url.Values{}
	q.Set("q", "service_instance_guid:"+guid)
	serviceKeys, err := c.ListServiceKeysByQuery(q)
	if err != nil {
		return serviceKey, err
	}
	if len(serviceKeys) == 0 {
		return serviceKey, fmt.Errorf("Unable to find service key for guid %s", guid)
	}
	return serviceKeys[0], nil
}

// GetServiceKeysByInstanceGuid returns the service keys for a service instance.
// If none are found, it returns an error.
func (c *Client) GetServiceKeysByInstanceGuid(guid string) ([]ServiceKey, error) {
	q := url.Values{}
	q.Set("q", "service_instance_guid:"+guid)
	serviceKeys, err := c.ListServiceKeysByQuery(q)
	if err != nil {
		return serviceKeys, err
	}
	if len(serviceKeys) == 0 {
		return serviceKeys, fmt.Errorf("Unable to find service key for guid %s", guid)
	}
	return serviceKeys, nil
}

// CreateServiceKey creates a service key from the request. If a service key
// exists already, it returns an error containing `CF-ServiceKeyNameTaken`
func (c *Client) CreateServiceKey(csr CreateServiceKeyRequest) (ServiceKey, error) {
	var serviceKeyResource ServiceKeyResource

	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(csr)
	if err != nil {
		return ServiceKey{}, err
	}
	req := c.NewRequestWithBody("POST", "/v2/service_keys", buf)
	resp, err := c.DoRequest(req)
	if err != nil {
		return ServiceKey{}, err
	}
	if resp.StatusCode != http.StatusCreated {
		return ServiceKey{}, fmt.Errorf("CF API returned with status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return ServiceKey{}, err
	}
	err = json.Unmarshal(body, &serviceKeyResource)
	if err != nil {
		return ServiceKey{}, err
	}

	return serviceKeyResource.Entity, nil
}
