// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestClusterInfo(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "testclusterinfo")
	if err != nil {
		t.Fatal(err)
	}
	p := f.Name()
	defer os.RemoveAll(p)

	ci := ClusterInfo{
		URIs:     []string{"http://localhost:5000"},
		Endpoint: "/ext/bc/abc",
		PID:      os.Getpid(),
	}
	if err := ci.Save(p); err != nil {
		t.Fatal(err)
	}

	ci2, err := LoadClusterInfo(p)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ci, ci2) {
		t.Fatalf("unexpected %+v, expected %+v", ci2, ci)
	}
}
