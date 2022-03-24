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
	Name          string       `xml:"name"`
	ArtifactId    string       `xml:"artifactId"`
	GroupId       string       `xml:"groupId"`
	ParentGroupId string       `xml:"parent>groupId"`
	Version       string       `xml:"version"`
	ParentVersion string       `xml:"parent>version"`
	Dependencies  []Dependency `xml:"dependencies>dependency"`
	Properties    Properties   `xml:"properties"`
	License       string       `xml:"licenses>license>name"`
	LicenseUrl    string       `xml:"licenses>license>url"`
}

func (p *Pom) GetGroupId() string {
	if p.GroupId != "" {
		return p.GroupId
	}
	return p.ParentGroupId
}

func (p *Pom) GetVersion() string {
	if p.Version != "" {
		return p.Version
	}
	return p.ParentVersion
}

type Dependency struct {
	ArtifactId string `xml:"artifactId"`
	GroupId    string `xml:"groupId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
}

type Properties struct {
	Property []Property `xml:",any"`
}

func (p *Properties) Find(name string) string {
	for _, prop := range p.Property {
		if "${"+prop.Name+"}" == name {
			return prop.Value
		}
	}
	return ""
}

type Property struct {
	Name  string
	Value string
}

func (v *Property) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	v.Name = start.Name.Local
	return d.DecodeElement(&v.Value, &start)
}

func (d *Dependency) JarPath() string {
	return fmt.Sprintf("/%s/%s/%s/%s-%s.jar", strings.ReplaceAll(d.GroupId, ".", "/"), d.ArtifactId, d.Version, d.ArtifactId, d.Version)
}

func (d *Dependency) PomPath() string {
	return fmt.Sprintf("/%s/%s/%s/%s-%s.pom", strings.ReplaceAll(d.GroupId, ".", "/"), d.ArtifactId, d.Version, d.ArtifactId, d.Version)
}

func TestSomething(t *testing.T) {
	d := Dependency{
		ArtifactId: "xs-jdbc-routing-datasource",
		GroupId:    "com.sap.cloud.sjb",
		Version:    "1.27.0",
	}

	dir, err := ioutil.TempDir("", "dependency-manager")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	err = ResolveFrom(d, dir)
	assert.NoError(t, err)
}

func ResolveFrom(dependency Dependency, dir string) error {
	fmt.Printf("ResolveFrom(%+v)\n", dependency)

	url := "https://repo1.maven.org/maven2" + dependency.JarPath()
	err := DownloadFile(filepath.Join(dir, filepath.Base(url)), url)
	if err != nil {
		return err
	}

	sha, err := Sha256(filepath.Join(dir, filepath.Base(url)))
	if err != nil {
		return err
	}

	//fmt.Println("getting pom...")
	pomUrl := "https://repo1.maven.org/maven2" + dependency.PomPath()
	pom, err := GetPom(pomUrl)
	if err != nil {
		return err
	}

	fmt.Println("  [[metadata.dependencies]]")
	fmt.Printf("    cpes = [\"cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*\"]\n", pom.GetGroupId(), pom.ArtifactId, pom.GetVersion())
	fmt.Printf("    id = \"%s\"\n", pom.ArtifactId)
	fmt.Printf("    name = \"%s\"\n", pom.Name)
	fmt.Printf("    purl = \"pkg:generic/%s@%s\"\n", pom.ArtifactId, pom.GetVersion())
	fmt.Printf("    sha256 = \"%s\"\n", sha)
	fmt.Println("    stacks = [\"io.buildpacks.stacks.bionic\", \"io.buildpacks.stacks.tiny\", \"*\"]")
	fmt.Printf("    uri = \"%s\"\n", url)
	fmt.Printf("    version = \"%s\"\n", pom.GetVersion())
	fmt.Println("")
	if pom.License != "" {
		fmt.Println("    [[metadata.dependencies.licenses]]")
		fmt.Printf("      type = \"%s\"\n", pom.License)
		if pom.LicenseUrl != "" {
			fmt.Printf("      uri = \"%s\"\n", pom.LicenseUrl)
		}
	}
	fmt.Println("")

	for _, d := range pom.Dependencies {
		if d.Scope != "test" {
			var dependency Dependency
			if strings.HasPrefix(d.Version, "${") {
				dependency = Dependency{
					ArtifactId: d.ArtifactId,
					GroupId:    d.GroupId,
					Version:    pom.Properties.Find(d.Version),
				}
			} else {
				dependency = Dependency{
					ArtifactId: d.ArtifactId,
					GroupId:    d.GroupId,
					Version:    d.Version,
				}
			}

			if dependency.Version != "" {
				err = ResolveFrom(dependency, dir)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
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
		return Pom{}, fmt.Errorf("status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Pom{}, fmt.Errorf("read body: %v", err)
	}

	var result Pom
	err = xml.Unmarshal(data, &result)
	if err != nil {
		return Pom{}, fmt.Errorf("parse: %v", err)
	}

	fmt.Println("pom=", result)

	return result, nil
}
