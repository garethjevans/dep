package dependency_test

import (
	"github.com/garethjevans/dep/dependency"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestSomething(t *testing.T) {
	d := dependency.Dependency{
		ArtifactId: "xs-jdbc-routing-datasource",
		GroupId:    "com.sap.cloud.sjb",
		Version:    "1.27.0",
	}

	dir, err := ioutil.TempDir("", "dependency-manager")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	err = dependency.ResolveFrom(d, dir, []string{"https://repo1.maven.org/maven2"})
	assert.NoError(t, err)
}
