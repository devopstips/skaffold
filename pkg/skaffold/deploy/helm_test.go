package deploy

import (
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

var testBuildResult = &build.BuildResult{
	Builds: []build.Build{
		{
			ImageName: "skaffold-helm",
			Tag:       "skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184",
		},
	},
}

var testDeployConfig = &config.DeployConfig{
	DeployType: config.DeployType{
		HelmDeploy: &config.HelmDeploy{
			Releases: []config.HelmRelease{
				{
					Name:      "skaffold-helm",
					ChartPath: "examples/test",
					Values: map[string]string{
						"image.tag": "skaffold-helm",
					},
				},
			},
		},
	},
}

var testDeployConfigParameterUnmatched = &config.DeployConfig{
	DeployType: config.DeployType{
		HelmDeploy: &config.HelmDeploy{
			Releases: []config.HelmRelease{
				{
					Name:      "skaffold-helm",
					ChartPath: "examples/test",
					Values: map[string]string{
						"image.tag": "skaffold-helm-unmatched",
					},
				},
			},
		},
	},
}

func TestNewHelmDeployerNoError(t *testing.T) {
	_, err := NewHelmDeployer(testDeployConfig)
	if err != nil {
		t.Errorf("Unexpected error new config: %s", err)
	}
}

func TestHelmDeploy(t *testing.T) {
	util.DefaultExecCommand = testutil.NewFakeRunCommand("", "", nil)
	defer util.ResetDefaultExecCommand()

	var tests = []struct {
		description string
		cmd         util.Command
		deployer    *HelmDeployer
		buildResult *build.BuildResult

		shouldErr bool
	}{
		{
			description: "deploy success",
			cmd:         &MockHelm{t: t},
			deployer: &HelmDeployer{
				DeployConfig: testDeployConfig,
			},
			buildResult: testBuildResult,
		},
		{
			description: "deploy error unmatched parameter",
			cmd:         &MockHelm{t: t},
			deployer: &HelmDeployer{
				DeployConfig: testDeployConfigParameterUnmatched,
			},
			buildResult: testBuildResult,
			shouldErr:   true,
		},
		{
			description: "get failure should install not upgrade",
			cmd: &MockHelm{
				t:             t,
				getResult:     cmdOutput{"", "", fmt.Errorf("not found")},
				upgradeResult: cmdOutput{"", "", fmt.Errorf("should not have called upgrade")},
			},
			deployer: &HelmDeployer{
				DeployConfig: testDeployConfig,
			},
			buildResult: testBuildResult,
		},
		{
			description: "get success should upgrade not install",
			cmd: &MockHelm{
				t:             t,
				getResult:     cmdOutput{"", "", fmt.Errorf("not found")},
				upgradeResult: cmdOutput{"", "", fmt.Errorf("should not have called install")},
			},
			deployer: &HelmDeployer{
				DeployConfig: testDeployConfig,
			},
			buildResult: testBuildResult,
		},
		{
			description: "deploy error",
			cmd: &MockHelm{
				t:             t,
				upgradeResult: cmdOutput{"", "", fmt.Errorf("unexpected error")},
			},
			shouldErr: true,
			deployer: &HelmDeployer{
				DeployConfig: testDeployConfig,
			},
			buildResult: testBuildResult,
		},
	}

	for _, tt := range tests {
		fmt.Println("test")
		t.Run(tt.description, func(t *testing.T) {
			util.DefaultExecCommand = tt.cmd
			_, err := tt.deployer.Run(tt.buildResult)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}

}

type MockHelm struct {
	getResult     cmdOutput
	installResult cmdOutput
	upgradeResult cmdOutput

	t *testing.T
}

type cmdOutput struct {
	stdout string
	stderr string
	err    error
}

func (c cmdOutput) out() ([]byte, []byte, error) {
	return []byte(c.stdout), []byte(c.stderr), c.err
}

func (m *MockHelm) RunCommand(c *exec.Cmd, _ io.Reader) ([]byte, []byte, error) {
	if len(c.Args) < 1 {
		m.t.Errorf("Not enough args in command %v", c)
	}
	fmt.Println("swithc")
	switch c.Args[1] {
	case "get":
		return m.getResult.out()
	case "install":
		fmt.Println("bah")
		return m.installResult.out()
	case "upgrade":
		return m.upgradeResult.out()
	}

	m.t.Errorf("Unknown helm command: %+v", c)
	return nil, nil, nil
}
