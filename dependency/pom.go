package dependency

import (
	"encoding/xml"
	"fmt"
	"strings"
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
