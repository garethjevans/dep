package pkg

import (
	"errors"
	"github.com/garethjevans/dep/dependency"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

func init() {
	rootCmd.AddCommand(NewAnalyzeCmd().Cmd)
}

type AnalyzeCmd struct {
	Cmd          *cobra.Command
	GroupId      string
	ArtifactId   string
	Version      string
	Repositories []string
}

func NewAnalyzeCmd() AnalyzeCmd {
	cmd := AnalyzeCmd{}
	cmd.Cmd = &cobra.Command{
		Use:     "analyze",
		Short:   "Analyze maven dependencies",
		Long:    ``,
		Aliases: []string{"a", "analyse"},
		Run: func(c *cobra.Command, args []string) {
			err := cmd.Run()
			if err != nil {
				log.Fatalf("Unable to run %s", err)
			}
		},
	}

	// func (f *FlagSet) StringVarP(p *string, name, shorthand string, value string, usage string) {
	cmd.Cmd.Flags().StringVarP(&cmd.GroupId, "groupId", "g", "", "groupId to search from")
	cmd.Cmd.Flags().StringVarP(&cmd.ArtifactId, "artifactId", "a", "", "artifactId to search from")
	cmd.Cmd.Flags().StringVarP(&cmd.Version, "version", "v", "", "version to search from")

	cmd.Cmd.Flags().StringArrayVarP(&cmd.Repositories, "repository", "r", []string{"https://repo1.maven.org/maven2"}, "repositories to search")

	return cmd
}

func (a *AnalyzeCmd) Run() error {
	if a.ArtifactId == "" {
		return errors.New("--artifactId must be set")
	}
	if a.GroupId == "" {
		return errors.New("--groupId must be set")
	}
	if a.Version == "" {
		return errors.New("--version must be set")
	}

	d := dependency.Dependency{
		GroupId:    a.GroupId,
		ArtifactId: a.ArtifactId,
		Version:    a.Version,
	}

	dir, err := ioutil.TempDir("", "dependency-manager")
	if err != nil {
		return err
	}

	err = dependency.ResolveFrom(d, dir, a.Repositories)
	if err != nil {
		return err
	}

	defer func() { os.RemoveAll(dir) }()

	return nil
}
