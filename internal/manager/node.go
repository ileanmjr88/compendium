// Copyright 2026 Ilean Monterrubio Jr
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type NodeManager struct {
	paths *env.Paths
}

func (m *NodeManager) Name() string {
	return "node"
}

func (m *NodeManager) EnvVars(version string, platform string) []EnvVar {
	return nil
}

func (m *NodeManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	return nil
}
