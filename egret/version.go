// Copyright (c) 2012-2016 The Eject Framework Authors, All rights reserved.
// Eject Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"github.com/kenorld/eject-core"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "displays the Eject Framework and Go version",
	Long: `
Displays the Eject Framework and Go version.

For example:

    eject version
`,
}

func init() {
	cmdVersion.Run = versionApp
}

func versionApp(args []string) {
	fmt.Printf("Version(s):")
	fmt.Printf("\n   Eject v%v (%v)", eject.Version, eject.BuildDate)
	fmt.Printf("\n   %s %s/%s\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
