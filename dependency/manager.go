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

var Counter int
var ExcludedGroupIds = []string{"org.apache.tomcat", "javax.servlet", "javax.mail", "org.springframework"}
var added []string

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
		break
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

	Counter++

	added = append(added, fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.ArtifactId))

	if len(dependency.Parents) > 0 {
		fmt.Printf("  # Parents %s\n", strings.Join(dependency.Parents, "=>"))
	}
	fmt.Printf("  # From %s\n", pomUrl)
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
		if !InList([]string{"test", "runtime"}, d.Scope) {
			if !InList(ExcludedGroupIds, d.GroupId) {
				var newDependency Dependency
				if strings.HasPrefix(d.Version, "${") {
					if d.Version == "${project.version}" {
						newDependency = Dependency{
							ArtifactId: d.ArtifactId,
							GroupId:    d.GroupId,
							Version:    pom.GetVersion(),
							Parents:    append(dependency.Parents, fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.ArtifactId)),
						}
					} else {
						newDependency = Dependency{
							ArtifactId: d.ArtifactId,
							GroupId:    d.GroupId,
							Version:    pom.Properties.Find(d.Version),
							Parents:    append(dependency.Parents, fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.ArtifactId)),
						}
					}
				} else {
					newDependency = Dependency{
						ArtifactId: d.ArtifactId,
						GroupId:    d.GroupId,
						Version:    d.Version,
						Parents:    append(dependency.Parents, fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.ArtifactId)),
					}
				}

				if !InList(added, fmt.Sprintf("%s:%s", newDependency.GroupId, newDependency.ArtifactId)) {
					if newDependency.Version != "" {
						err = ResolveFrom(newDependency, dir, repositories)
						if err != nil {
							return err
						}
					} else {
						fmt.Printf("  # unable to add %s:%s:%s\n", d.GroupId, d.ArtifactId, d.Version)
					}
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

func InList(in []string, test string) bool {
	for _, i := range in {
		if test == i {
			return true
		}
	}
	return false
}
