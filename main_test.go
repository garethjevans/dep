package dep_test

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type Pom struct {
	Name       string `xml:"name"`
	ArtifactId string `xml:"artifactId"`
	GroupId    string `xml:"groupId"`
	Version    string `xml:"version"`
}

func TestSomething(t *testing.T) {
	//url := "https://repo1.maven.org/maven2/com/sap/cloud/sjb/xs-env/1.24.0/xs-env-1.24.0.jar"
	url := "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-core/2.13.2/jackson-core-2.13.2.jar"

	dir, err := ioutil.TempDir("", "dependency-manager")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	err = DownloadFile(filepath.Join(dir, filepath.Base(url)), url)
	assert.NoError(t, err)

	sha, err := Sha256(filepath.Join(dir, filepath.Base(url)))
	assert.NoError(t, err)

	pom, err := GetPom(strings.TrimSuffix(url, ".jar") + ".pom")
	assert.NoError(t, err)

	fmt.Println("  [[metadata.dependencies]]")
	fmt.Printf("    cpes = [\"cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*\"]\n", pom.GroupId, pom.ArtifactId, pom.Version)
	fmt.Printf("    id = \"%s\"\n", pom.ArtifactId)
	fmt.Printf("    name = \"%s\"\n", pom.Name)
	fmt.Printf("    purl = \"pkg:generic/%s@%s\"\n", pom.ArtifactId, pom.Version)
	fmt.Printf("    sha256 = \"%s\"\n", sha)
	fmt.Println("    stacks = [\"io.buildpacks.stacks.bionic\", \"io.buildpacks.stacks.tiny\"]")
	fmt.Printf("    uri = \"%s\"\n", url)
	fmt.Printf("    version = \"%s\"\n", pom.Version)
	fmt.Println("")
	fmt.Println("    [[metadata.dependencies.licenses]]")
	fmt.Println("    type = \"BSD-2-Clause\"")
	//fmt.Println("    uri = \"https://jdbc.postgresql.org/about/license.html\"")
}

func Sha256(filepath string) (string, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("writing", filepath)
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func GetPom(url string) (Pom, error) {
	resp, err := http.Get(url)
	if err != nil {
		return Pom{}, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Pom{}, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Pom{}, fmt.Errorf("Read body: %v", err)
	}

	var result Pom
	err = xml.Unmarshal(data, &result)
	if err != nil {
		return Pom{}, fmt.Errorf("Read body: %v", err)
	}

	return result, nil
}
