// Copyright (c) 2012-2016 The Egret Framework Authors, All rights reserved.
// Egret Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"github.com/kenorld/egret-core"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "displays the Egret Framework and Go version",
	Long: `
Displays the Egret Framework and Go version.

For example:

    egret version
`,
}

func init() {
	cmdVersion.Run = versionApp
}

func versionApp(args []string) {
	fmt.Printf("Version(s):")
	fmt.Printf("\n   Egret v%v (%v)", egret.Version, egret.BuildDate)
	fmt.Printf("\n   %s %s/%s\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
