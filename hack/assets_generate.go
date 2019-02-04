// +build ignore

package main

import (
	"log"

	"github.com/Ridecell/ridectl/pkg/cmd"
	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(cmd.Templates, vfsgen.Options{
		PackageName:  "cmd",
		BuildTags:    "release",
		VariableName: "Templates",
		Filename:     "zz_generated.templates.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
