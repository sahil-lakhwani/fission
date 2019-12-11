package v1

import (
	"io/ioutil"
	"net/http"

	ferror "github.com/fission/fission/pkg/error"

	"github.com/fission/fission/pkg/controller/client/rest"
)

type (
	V1Interface interface {
		MiscGetter
		CanaryConfigGetter
		EnvironmentGetter
		FunctionGetter
		HTTPTriggerGetter
		KubeWatcherGetter
		MessageQueueTriggerGetter
		PackageGetter
		TimeTriggerGetter
	}

	V1 struct {
		restClient rest.Interface
	}
)

func MakeV1Client(restClient rest.Interface) *V1 {
	return &V1{restClient: restClient}
}

func (c *V1) Misc() MiscInterface {
	return newMiscClient(c)
}

func (c *V1) CanaryConfig() CanaryConfigInterface {
	return newCanaryConfigClient(c)
}

func (c *V1) Environment() EnvironmentInterface {
	return newEnvironmentClient(c)
}

func (c *V1) Function() FunctionInterface {
	return newFunctionClient(c)
}

func (c *V1) HTTPTrigger() HTTPTriggerInterface {
	return newHTTPTriggerClient(c)
}

func (c *V1) KubeWatcher() KubeWatcherInterface {
	return newKubeWatcher(c)
}

func (c *V1) MessageQueueTrigger() MessageQueueTriggerInterface {
	return newMessageQueueTrigger(c)
}

func (c *V1) Package() PackageInterface {
	return newPackageClient(c)
}

func (c *V1) TimeTrigger() TimeTriggerInterface {
	return newTimeTriggerClient(c)
}

func handleResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != 200 {
		return nil, ferror.MakeErrorFromHTTP(resp)
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func handleCreateResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != 201 {
		return nil, ferror.MakeErrorFromHTTP(resp)
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}
