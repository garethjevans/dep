package dependency

import (
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ResolveFrom(dependency Dependency, dir string, repositories []string) error {
	//fmt.Printf("ResolveFrom(%+v)\n", dependency)

	var url string
	var selectedRepository string

	for _, repository := range repositories {
		url = repository + dependency.JarPath()

		//fmt.Println("checking", url)

		err := DownloadFile(filepath.Join(dir, filepath.Base(url)), url)
		if err != nil {
			//fmt.Println("Unable to access ", url)
			continue
		}
		selectedRepository = repository
	}

	sha, err := Sha256(filepath.Join(dir, filepath.Base(url)))
	if err != nil {
		return err
	}

	//fmt.Println("getting pom...")
	pomUrl := selectedRepository + dependency.PomPath()
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
				err = ResolveFrom(dependency, dir, repositories)
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

	if resp.StatusCode != 200 {
		return errors.New("unexpected error code" + resp.Status)
	}

	// Create the file
	//fmt.Println("downloading to ", filepath)
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

	//fmt.Println("pom=", result)

	return result, nil
}
