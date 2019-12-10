/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
)

type (
	HTTPTriggerGetter interface {
		HTTPTrigger() HTTPTriggerInterface
	}

	HTTPTriggerInterface interface {
		Create(t *fv1.HTTPTrigger) (*metav1.ObjectMeta, error)
		Get(m *metav1.ObjectMeta) (*fv1.HTTPTrigger, error)
		Update(t *fv1.HTTPTrigger) (*metav1.ObjectMeta, error)
		Delete(m *metav1.ObjectMeta) error
		List(triggerNamespace string) ([]fv1.HTTPTrigger, error)
	}
)

func (c *Client) HTTPTriggerCreate(t *fv1.HTTPTrigger) (*metav1.ObjectMeta, error) {
	err := t.Validate()
	if err != nil {
		return nil, fv1.AggregateValidationErrors("HTTPTrigger", err)
	}

	reqbody, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	resp, err := c.create("triggers/http", "application/json", reqbody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleCreateResponse(resp)
	if err != nil {
		return nil, err
	}

	var m metav1.ObjectMeta
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (c *Client) HTTPTriggerGet(m *metav1.ObjectMeta) (*fv1.HTTPTrigger, error) {
	relativeUrl := fmt.Sprintf("triggers/http/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)

	resp, err := c.get(relativeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	var t fv1.HTTPTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) HTTPTriggerUpdate(t *fv1.HTTPTrigger) (*metav1.ObjectMeta, error) {
	err := t.Validate()
	if err != nil {
		return nil, fv1.AggregateValidationErrors("HTTPTrigger", err)
	}

	reqbody, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	relativeUrl := fmt.Sprintf("triggers/http/%v", t.Metadata.Name)

	resp, err := c.put(relativeUrl, "application/json", reqbody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	var m metav1.ObjectMeta
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) HTTPTriggerDelete(m *metav1.ObjectMeta) error {
	relativeUrl := fmt.Sprintf("triggers/http/%v", m.Name)
	relativeUrl += fmt.Sprintf("?namespace=%v", m.Namespace)
	return c.delete(relativeUrl)
}

func (c *Client) HTTPTriggerList(triggerNamespace string) ([]fv1.HTTPTrigger, error) {
	relativeUrl := fmt.Sprintf("triggers/http?namespace=%v", triggerNamespace)
	resp, err := c.get(relativeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := c.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	triggers := make([]fv1.HTTPTrigger, 0)
	err = json.Unmarshal(body, &triggers)
	if err != nil {
		return nil, err
	}

	return triggers, nil
}
