//go:build tools
// +build tools

package tools

import (
	_ "github.com/openshift/api/tools"
	_ "github.com/openshift/api/tools/codegen/cmd"
)
