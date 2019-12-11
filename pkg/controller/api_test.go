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

package controller

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/controller/client"
	ferror "github.com/fission/fission/pkg/error"
)

var g struct {
	cmd.CommandActioner
}

func panicIf(err error) {
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

func assert(c bool, msg string) {
	if !c {
		log.Fatalf("assert failed: %v", msg)
	}
}

func assertNameReuseFailure(err error, name string) {
	assert(err != nil, "recreating "+name+" with same name must fail")
	fe, ok := err.(ferror.Error)
	assert(ok, "error must be a fission Error")
	assert(fe.Code == ferror.ErrorNameExists, "error must be a name exists error")
}

func assertNotFoundFailure(err error, name string) {
	assert(err != nil, "requesting a non-existent "+name+" must fail")
	fe, ok := err.(ferror.Error)
	assert(ok, "error must be a fission Error")
	if fe.Code != ferror.ErrorNotFound {
		log.Fatalf("error must be a not found error: %v", fe)
	}
}

func assertCronSpecFails(err error) {
	assert(err != nil, "using an invalid cron spec must fail")
	ok := strings.Contains(err.Error(), "not a valid cron spec")
	assert(ok, "invalid cron spec must fail")
}

func TestFunctionApi(t *testing.T) {
	testFunc := &fv1.Function{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fv1.FunctionSpec{
			Environment: fv1.EnvironmentReference{
				Name:      "nodejs",
				Namespace: metav1.NamespaceDefault,
			},
			Package: fv1.FunctionPackageRef{
				FunctionName: "xxx",
				PackageRef: fv1.PackageRef{
					Namespace:       metav1.NamespaceDefault,
					Name:            "xxx",
					ResourceVersion: "12345",
				},
			},
		},
	}
	_, err := g.client.V1().Function().Get(&metav1.ObjectMeta{
		Name:      testFunc.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "function")

	m, err := g.client.V1().Function().Create(testFunc)
	panicIf(err)
	defer func() {
		err := g.client.V1().Function().Delete(m)
		panicIf(err)
	}()

	_, err = g.client.V1().Function().Create(testFunc)
	assertNameReuseFailure(err, "function")

	testFunc.Metadata.ResourceVersion = m.ResourceVersion
	testFunc.Spec.Package.FunctionName = "yyy"
	_, err = g.client.V1().Function().Update(testFunc)
	panicIf(err)

	testFunc.Metadata.ResourceVersion = ""
	testFunc.Metadata.Name = "bar"
	m2, err := g.client.V1().Function().Create(testFunc)
	panicIf(err)
	defer g.client.V1().Function().Delete(m2)

	funcs, err := g.client.V1().Function().List(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(funcs) == 2, fmt.Sprintf("created two functions, but found %v", len(funcs)))

	funcs_url := g.client.ServerURL() + "/v2/functions"
	resp, err := http.Get(funcs_url)
	panicIf(err)
	defer resp.Body.Close()
	assert(resp.StatusCode == 200, "http get status code on /v1/functions")

	var found bool = false
	for _, b := range resp.Header["Content-Type"] {
		if b == "application/json; charset=utf-8" {
			found = true
		}
	}
	assert(found, "incorrect response content type")
}

func TestHTTPTriggerApi(t *testing.T) {
	testTrigger := &fv1.HTTPTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fv1.HTTPTriggerSpec{
			Method:      http.MethodGet,
			RelativeURL: "/hello",
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}
	_, err := g.client.V1().HTTPTrigger().Get(&metav1.ObjectMeta{
		Name:      testTrigger.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "httptrigger")

	m, err := g.client.V1().HTTPTrigger().Create(testTrigger)
	panicIf(err)
	defer g.client.V1().HTTPTrigger().Delete(m)

	_, err = g.client.V1().HTTPTrigger().Create(testTrigger)
	assertNameReuseFailure(err, "httptrigger")

	tr, err := g.client.V1().HTTPTrigger().Get(m)
	panicIf(err)
	assert(testTrigger.Spec.Method == tr.Spec.Method &&
		testTrigger.Spec.RelativeURL == tr.Spec.RelativeURL &&
		testTrigger.Spec.FunctionReference.Type == tr.Spec.FunctionReference.Type &&
		testTrigger.Spec.FunctionReference.Name == tr.Spec.FunctionReference.Name, "trigger should match after reading")

	testTrigger.Metadata.ResourceVersion = m.ResourceVersion
	testTrigger.Spec.RelativeURL = "/hi"
	_, err = g.client.V1().HTTPTrigger().Update(testTrigger)
	panicIf(err)

	testTrigger.Metadata.ResourceVersion = ""
	testTrigger.Metadata.Name = "yyy"
	_, err = g.client.V1().HTTPTrigger().Create(testTrigger)
	assert(err != nil, "duplicate trigger should not be allowed")

	testTrigger.Spec.RelativeURL = "/hi2"
	m2, err := g.client.V1().HTTPTrigger().Create(testTrigger)
	panicIf(err)
	defer g.client.V1().HTTPTrigger().Delete(m2)

	ts, err := g.client.V1().HTTPTrigger().List(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 2, fmt.Sprintf("created two triggers, but found %v", len(ts)))
}

func TestEnvironmentApi(t *testing.T) {

	testEnv := &fv1.Environment{
		Metadata: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fv1.EnvironmentSpec{
			Version: 1,
			Runtime: fv1.Runtime{
				Image: "gcr.io/xyz",
			},
			Resources: v1.ResourceRequirements{},
		},
	}
	_, err := g.client.V1().Environment().Get(&metav1.ObjectMeta{
		Name:      testEnv.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "environment")

	m, err := g.client.V1().Environment().Create(testEnv)
	panicIf(err)
	defer g.client.V1().Environment().Delete(m)

	_, err = g.client.V1().Environment().Create(testEnv)
	assertNameReuseFailure(err, "environment")

	e, err := g.client.V1().Environment().Get(m)
	panicIf(err)
	assert(reflect.DeepEqual(testEnv.Spec, e.Spec), "env should match after reading")

	testEnv.Metadata.ResourceVersion = m.ResourceVersion
	testEnv.Spec.Runtime.Image = "another-img"
	_, err = g.client.V1().Environment().Update(testEnv)
	panicIf(err)

	testEnv.Metadata.ResourceVersion = ""
	testEnv.Metadata.Name = "bar"

	m2, err := g.client.V1().Environment().Create(testEnv)
	panicIf(err)
	defer g.client.V1().Environment().Delete(m2)

	ts, err := g.client.V1().Environment().List(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 2, fmt.Sprintf("created two envs, but found %v", len(ts)))
}

func TestWatchApi(t *testing.T) {
	testWatch := &fv1.KubernetesWatchTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "xxx",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fv1.KubernetesWatchTriggerSpec{
			Namespace: "default",
			Type:      "pod",
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: "foo",
			},
		},
	}
	_, err := g.client.V1().KubeWatcher().Get(&metav1.ObjectMeta{
		Name:      testWatch.Metadata.Name,
		Namespace: metav1.NamespaceDefault,
	})
	assertNotFoundFailure(err, "watch")

	m, err := g.client.V1().KubeWatcher().Create(testWatch)
	panicIf(err)
	defer g.client.V1().KubeWatcher().Delete(m)

	_, err = g.client.V1().KubeWatcher().Create(testWatch)
	assertNameReuseFailure(err, "watch")

	w, err := g.client.V1().KubeWatcher().Get(m)
	panicIf(err)
	assert(testWatch.Spec.Namespace == w.Spec.Namespace &&
		testWatch.Spec.Type == w.Spec.Type &&
		testWatch.Spec.FunctionReference.Type == w.Spec.FunctionReference.Type &&
		testWatch.Spec.FunctionReference.Name == w.Spec.FunctionReference.Name, "watch should match after reading")

	testWatch.Metadata.Name = "yyy"
	m2, err := g.client.V1().KubeWatcher().Create(testWatch)
	panicIf(err)
	defer g.client.V1().KubeWatcher().Delete(m2)

	ws, err := g.client.V1().KubeWatcher().List(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ws) == 2, fmt.Sprintf("created two watches, but found %v", len(ws)))
}

func TestTimeTriggerApi(t *testing.T) {
	testTrigger := &fv1.TimeTrigger{
		Metadata: metav1.ObjectMeta{
			Name:      "xxx",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: fv1.TimeTriggerSpec{
			Cron: "0 30 * * * *",
			FunctionReference: fv1.FunctionReference{
				Type: fv1.FunctionReferenceTypeFunctionName,
				Name: "asdf",
			},
		},
	}
	_, err := g.client.V1().TimeTrigger().Get(&metav1.ObjectMeta{Name: testTrigger.Metadata.Name})
	assertNotFoundFailure(err, "trigger")

	m, err := g.client.V1().TimeTrigger().Create(testTrigger)
	panicIf(err)
	defer g.client.V1().TimeTrigger().Delete(m)

	_, err = g.client.V1().TimeTrigger().Create(testTrigger)
	assertNameReuseFailure(err, "trigger")

	tr, err := g.client.V1().TimeTrigger().Get(m)
	panicIf(err)
	assert(testTrigger.Spec.Cron == tr.Spec.Cron &&
		testTrigger.Spec.FunctionReference.Type == tr.Spec.FunctionReference.Type &&
		testTrigger.Spec.FunctionReference.Name == tr.Spec.FunctionReference.Name, "trigger should match after reading")

	testTrigger.Metadata.ResourceVersion = m.ResourceVersion
	testTrigger.Spec.Cron = "@hourly"
	_, err = g.client.V1().TimeTrigger().Update(testTrigger)
	panicIf(err)

	testTrigger.Metadata.ResourceVersion = ""
	testTrigger.Metadata.Name = "yyy"
	testTrigger.Spec.Cron = "Not valid cron spec"
	_, err = g.client.V1().TimeTrigger().Create(testTrigger)
	assertCronSpecFails(err)

	ts, err := g.client.V1().TimeTrigger().List(metav1.NamespaceDefault)
	panicIf(err)
	assert(len(ts) == 1, fmt.Sprintf("created two time triggers, but found %v", len(ts)))
}

func TestMain(m *testing.M) {
	flag.Parse()

	// skip test if no cluster available for testing
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) == 0 {
		log.Println("Skipping test, no kubernetes cluster")
		return
	}

	logger, err := zap.NewDevelopment()
	panicIf(err)

	go Start(logger, 8888, true)

	time.Sleep(5 * time.Second)
	g.client = client.MakeClientset("http://localhost:8888")

	resp, err := http.Get("http://localhost:8888/")
	panicIf(err)
	assert(resp.StatusCode == 200, "http get status code on root")

	var found bool = false
	for _, b := range resp.Header["Content-Type"] {
		if b == "application/json; charset=utf-8" {
			found = true
		}
	}
	assert(found, "incorrect response content type")

	_, err = ioutil.ReadAll(resp.Body)
	panicIf(err)

	os.Exit(m.Run())
}
