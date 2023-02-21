/*
Copyright 2023 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/ssh"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
)

const (
	giteaImageName    = "gitea/gitea:1.17"
	hostaliasesEnvKey = "HOSTALIASES"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "boostrap-git-test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	hostAliases := filepath.Join(tmpDir, ".hosts")
	f, err := os.Create(hostAliases)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	f.Close()
	err = os.Setenv(hostaliasesEnvKey, hostAliases)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	exitVal := m.Run()
	os.Exit(exitVal)
	os.Unsetenv(hostaliasesEnvKey)
}

func TestBootstrapGit_InvalidKubernetesConfiguration(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config:      bootstrapGitInvalidKubernetesConfiguration(),
				ExpectError: regexp.MustCompile("Expected configured provider clients."),
			},
		},
	})
}

func TestBootstrapGit_InvalidCustomization(t *testing.T) {
	kustomizationOverride := `
kind: Kustomization
resources:
  - gotk-components.yaml`
	env := environment{
		httpClone: "https://gitub.com",
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config:      bootstrapGitCustomization(env, kustomizationOverride),
				ExpectError: regexp.MustCompile("Kustomization resource must contain: gotk-sync.yaml"),
			},
		},
	})
}

func TestAccBootstrapGit_HTTP(t *testing.T) {
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config: bootstrapGitHTTP(env),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
			{
				Config:            bootstrapGitHTTP(env),
				ResourceName:      "flux_bootstrap_git.this",
				ImportState:       true,
				ImportStateId:     "flux-system",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBootstrapGit_SSH(t *testing.T) {
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config: bootstrapGitSSH(env),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
			{
				Config:            bootstrapGitSSH(env),
				ResourceName:      "flux_bootstrap_git.this",
				ImportState:       true,
				ImportStateId:     "flux-system",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBootstrapGit_Drift(t *testing.T) {
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			// Basic installation of Flux
			{
				Config: bootstrapGitHTTP(env),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
			// Remove file and expect Terraform to detect drift.
			{
				PreConfig: func() {
					gitClient := getTestGitClient(t, env.username, env.password)
					_, err := gitClient.Clone(context.TODO(), env.httpClone, repository.CloneOptions{CheckoutStrategy: repository.CheckoutStrategy{Branch: defaultBranch}})
					require.NoError(t, err)
					os.Remove(filepath.Join(gitClient.Path(), "flux-system/kustomization.yaml"))
					_, err = gitClient.Commit(git.Commit{})
					require.NoError(t, err)
					err = gitClient.Push(context.TODO())
					require.NoError(t, err)
				},
				Config: bootstrapGitHTTP(env),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
			// Change path and expect files to be moved
			{
				Config: bootstrapGitCustomPath(env, "custom-path"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.custom-path/flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.custom-path/flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.custom-path/flux-system/gotk-sync.yaml"),
				),
			},
		},
	})
}

func TestAccBootstrapGit_Upgrade(t *testing.T) {
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config: bootstrapGitVersion(env, "v0.34.0"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
			{
				Config: bootstrapGitVersion(env, "v0.35.0"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
		},
	})
}

func TestAccBootstrapGit_Components(t *testing.T) {
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config: bootstrapGitComponents(env),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
				),
			},
		},
	})
}

func TestAccBootstrapGit_Customization(t *testing.T) {
	kustomizationOverride := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: all
      spec:
        template:
          metadata:
            annotations:
              cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
    target:
      kind: Deployment
      labelSelector: app.kubernetes.io/part-of=flux`
	env := setupEnvironment(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"flux": providerserver.NewProtocol6WithError(New("dev")()),
		},
		Steps: []resource.TestStep{
			{
				Config: bootstrapGitCustomization(env, kustomizationOverride),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttrSet("flux_bootstrap_git.this", "repository_files.flux-system/gotk-sync.yaml"),
					//resource.TestCheckResourceAttr("flux_bootstrap_git.this", "repository_files.flux-system/kustomization.yaml", kustomizationOverride),
					func(state *terraform.State) error {
						cfg, err := clientcmd.BuildConfigFromFlags("", env.kubeCfgPath)
						if err != nil {
							return nil
						}
						client, err := kubernetes.NewForConfig(cfg)
						if err != nil {
							return nil
						}
						deploymentList, err := client.AppsV1().Deployments("flux-system").List(context.TODO(), metav1.ListOptions{})
						if err != nil {
							return nil
						}
						for _, deployment := range deploymentList.Items {
							v, ok := deployment.Spec.Template.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"]
							if !ok {
								return fmt.Errorf("expected annotation not set in Deployment %s", deployment.Name)
							}
							if v != "true" {
								return fmt.Errorf("expected annotation value to be true but was %s", v)
							}
						}
						return nil
					},
				),
			},
		},
	})
}

func bootstrapGitInvalidKubernetesConfiguration() string {
	return `
    provider "flux" {}

    resource "flux_bootstrap_git" "this" {
      url = "https://example.com"
    }
  `
}

func bootstrapGitHTTP(env environment) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      http = {
        username = "%s"
        password = "%s"
        allow_insecure_http = true
      }
    }
	`, env.kubeCfgPath, env.httpClone, env.username, env.password)
}

func bootstrapGitSSH(env environment) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      ssh = {
        username = "git"
        private_key = <<EOF
%s
EOF
      }
    }
	`, env.kubeCfgPath, env.sshClone, env.privateKey)
}

func bootstrapGitCustomPath(env environment, path string) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      http = {
        username = "%s"
        password = "%s"
        allow_insecure_http = true
      }
      path = "%s"
    }
	`, env.kubeCfgPath, env.httpClone, env.username, env.password, path)
}

func bootstrapGitVersion(env environment, version string) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      http = {
        username = "%s"
        password = "%s"
        allow_insecure_http = true
      }
      version = "%s"
    }
	`, env.kubeCfgPath, env.httpClone, env.username, env.password, version)
}

func bootstrapGitCustomization(env environment, kustomizationOverride string) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      http = {
        username = "%s"
        password = "%s"
        allow_insecure_http = true
      }
      kustomization_override = <<EOT
%s
      EOT
    }
	`, env.kubeCfgPath, env.httpClone, env.username, env.password, kustomizationOverride)
}

func bootstrapGitComponents(env environment) string {
	return fmt.Sprintf(`
    provider "flux" {
      config_path = "%s"
    }

    resource "flux_bootstrap_git" "this" {
      url = "%s"
      http = {
        username = "%s"
        password = "%s"
        allow_insecure_http = true
      }
	  components           = [
        "helm-controller",
        "kustomize-controller",
        "notification-controller",
        "source-controller",
      ]
      components_extra     = [
        "image-automation-controller",
        "image-reflector-controller",
      ]
    }
	`, env.kubeCfgPath, env.httpClone, env.username, env.password)
}

type environment struct {
	kubeCfgPath string
	httpClone   string
	sshClone    string
	username    string
	password    string
	privateKey  string
}

func setupEnvironment(t *testing.T) environment {
	t.Helper()
	rand.Seed(time.Now().UnixNano())
	httpPort := rand.Intn(65535-1024) + 1024
	sshPort := httpPort + 10
	randSuffix := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	giteaName := fmt.Sprintf("gitea-%s", randSuffix)
	username := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	password := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	tmpDir := t.TempDir()
	giteaUrl := fmt.Sprintf("http://%s:%d", giteaName, httpPort)

	// Add entry to host aliases
	hostAliases := os.Getenv(hostaliasesEnvKey)
	f, err := os.OpenFile(hostAliases, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%s localhost\n", giteaName))
	require.NoError(t, err)

	// Run Gitea server
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()
	reader, err := cli.ImagePull(context.TODO(), giteaImageName, dockertypes.ImagePullOptions{})
	require.NoError(t, err)
	defer reader.Close()
	io.Copy(io.Discard, reader)

	portSet, portMap, err := nat.ParsePortSpecs([]string{fmt.Sprintf("127.0.0.1:%d:%d", httpPort, httpPort), fmt.Sprintf("127.0.0.1:%d:%d", sshPort, sshPort)})
	require.NoError(t, err)
	containerCfg := &container.Config{
		Image:        giteaImageName,
		ExposedPorts: portSet,
		Env: []string{
			"INSTALL_LOCK=true",
			fmt.Sprintf("GITEA__SERVER__ROOT_URL=%s", giteaUrl),
			fmt.Sprintf("GITEA__SERVER__HTTP_PORT=%d", httpPort),
			fmt.Sprintf("GITEA__SERVER__SSH_PORT=%d", sshPort),
			fmt.Sprintf("GITEA__SERVER__SSH_LISTEN_PORT=%d", sshPort),
			"GITEA__SERVER__START_SSH_SERVER=true",
		},
	}
	hostCfg := &container.HostConfig{
		PortBindings: portMap,
	}
	resp, err := cli.ContainerCreate(context.TODO(), containerCfg, hostCfg, nil, nil, giteaName)
	require.NoError(t, err)
	err = cli.ContainerStart(context.TODO(), resp.ID, dockertypes.ContainerStartOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		cli.ContainerRemove(context.TODO(), resp.ID, dockertypes.ContainerRemoveOptions{Force: true})
	})

	// Start Kind cluster
	kubeCfgPath := filepath.Join(tmpDir, ".kube", "config")
	p := cluster.NewProvider(cluster.ProviderWithDocker())
	err = p.Create(randSuffix, cluster.CreateWithKubeconfigPath(kubeCfgPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		p.Delete(randSuffix, kubeCfgPath)
	})
	networks, err := cli.NetworkList(context.TODO(), dockertypes.NetworkListOptions{Filters: filters.NewArgs(filters.Arg("name", "kind"))})
	require.NoError(t, err)
	require.Len(t, networks, 1)
	err = cli.NetworkConnect(context.TODO(), networks[0].ID, resp.ID, nil)
	require.NoError(t, err)

	// Create admin user in gitea
	// TODO: Need a better solution than just sleeping
	time.Sleep(1 * time.Second)
	execCfg := dockertypes.ExecConfig{
		User: "git",
		Cmd:  []string{"gitea", "admin", "user", "create", "--username", "gitea_admin", "--password", "foobar", "--email", "admin@example.com", "--admin"},
	}
	exec, err := cli.ContainerExecCreate(context.TODO(), resp.ID, execCfg)
	require.NoError(t, err)
	err = cli.ContainerExecStart(context.TODO(), exec.ID, dockertypes.ExecStartCheck{})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Create Gitea user
	giteaClient, err := gitea.NewClient(giteaUrl, gitea.SetBasicAuth("gitea_admin", "foobar"))
	require.NoError(t, err)
	mustChangePassword := false
	createUserOpt := gitea.CreateUserOption{
		Username:           username,
		Email:              "example@example.com",
		Password:           password,
		MustChangePassword: &mustChangePassword,
	}
	_, _, err = giteaClient.AdminCreateUser(createUserOpt)
	require.NoError(t, err)
	createRepoOpt := gitea.CreateRepoOption{
		Name:          randSuffix,
		AutoInit:      true,
		DefaultBranch: defaultBranch,
		Private:       true,
	}
	repo, _, err := giteaClient.AdminCreateRepo(username, createRepoOpt)
	require.NoError(t, err)

	keyPair, err := ssh.GenerateKeyPair(ssh.ECDSA_P256)
	require.NoError(t, err)
	createPublicKeyOpt := gitea.CreateKeyOption{
		Title:    "Key",
		Key:      string(keyPair.PublicKey),
		ReadOnly: false,
	}
	giteaClient.AdminCreateUserPublicKey(username, createPublicKeyOpt)

	return environment{
		kubeCfgPath: kubeCfgPath,
		httpClone:   repo.CloneURL,
		sshClone:    fmt.Sprintf("ssh://git@%s:%d/%s/%s.git", giteaName, sshPort, username, randSuffix),
		username:    username,
		password:    password,
		privateKey:  string(keyPair.PrivateKey),
	}
}

func getTestGitClient(t *testing.T, username, password string) *gogit.Client {
	t.Helper()
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	require.NoError(t, err)
	authOpts := git.AuthOptions{
		Transport: git.HTTP,
		Username:  username,
		Password:  password,
	}
	gitClient, err := gogit.NewClient(tmpDir, &authOpts, gogit.WithDiskStorage(), gogit.WithInsecureCredentialsOverHTTP())
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(gitClient.Path())
	})
	return gitClient
}
