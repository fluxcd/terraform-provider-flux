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
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/fluxcd/flux2/pkg/bootstrap"
	"github.com/fluxcd/flux2/pkg/log"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

type providerResourceData struct {
	rcg *utils.RESTClientGetter
	git *Git
}

func NewProviderResourceData(ctx context.Context, data ProviderModel) (*providerResourceData, error) {
	clientCfg, err := getClientConfiguration(ctx, data.Kubernetes)
	if err != nil {
		return nil, fmt.Errorf("Invalid Kubernetes configuration: %w", err)
	}
	rcg := utils.NewRestClientGetter(clientCfg)
	return &providerResourceData{
		rcg: rcg,
		git: data.Git,
	}, nil

}

func (prd *providerResourceData) GetKubernetesClient() (client.WithWatch, error) {
	kubeClient, err := utils.KubeClient(prd.rcg, &runclient.Options{})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func (prd *providerResourceData) GetGitClient(ctx context.Context) (*gogit.Client, error) {
	// Git configuration
	authOpts, err := getAuthOpts(prd.git)
	if err != nil {
		return nil, err
	}
	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage()}
	if prd.git.Http != nil && prd.git.Http.InsecureHttpAllowed.ValueBool() {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
	}

	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary working directory for git repository: %w", err)
	}
	client, err := gogit.NewClient(tmpDir, authOpts, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create git client: %w", err)
	}
	// TODO: Need to conditionally clone here. If repository is empty this will fail.
	_, err = client.Clone(ctx, prd.GetRepositoryURL().String(), repository.CloneOptions{CheckoutStrategy: repository.CheckoutStrategy{Branch: prd.git.Branch.ValueString()}})
	if err != nil {
		return nil, fmt.Errorf("could not clone git repository: %w", err)
	}
	return client, nil
}

func (prd *providerResourceData) GetBootstrapOptions() ([]bootstrap.GitOption, error) {
	entityList, err := prd.GetEntityList()
	if err != nil {
		return nil, err
	}
	return []bootstrap.GitOption{
		bootstrap.WithRepositoryURL(prd.GetRepositoryURL().String()),
		bootstrap.WithKubeconfig(prd.rcg, &runclient.Options{}),
		bootstrap.WithBranch(prd.git.Branch.ValueString()),
		bootstrap.WithSignature(prd.git.AuthorName.ValueString(), prd.git.AuthorEmail.ValueString()),
		bootstrap.WithCommitMessageAppendix(prd.git.CommitMessageAppendix.ValueString()),
		bootstrap.WithGitCommitSigning(entityList, prd.git.GpgPassphrase.ValueString(), prd.git.GpgKeyID.ValueString()),
		bootstrap.WithLogger(log.NopLogger{}),
	}, nil
}

func (prd *providerResourceData) GetSecretOptions(secretName, namespace, targetPath string) (sourcesecret.Options, error) {
	secretOpts := sourcesecret.Options{
		Name:         secretName,
		Namespace:    namespace,
		TargetPath:   targetPath,
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if prd.git.Http != nil {
		secretOpts.Username = prd.git.Http.Username.ValueString()
		secretOpts.Password = prd.git.Http.Password.ValueString()
		secretOpts.CAFile = []byte(prd.git.Http.CertificateAuthority.ValueString())
	}
	if prd.git.Ssh != nil {
		if prd.git.Ssh.PrivateKey.ValueString() != "" {
			keypair, err := sourcesecret.LoadKeyPair([]byte(prd.git.Ssh.PrivateKey.ValueString()), prd.git.Ssh.Password.ValueString())
			if err != nil {
				return sourcesecret.Options{}, fmt.Errorf("Failed to load SSH Key Pair: %w", err)
			}
			secretOpts.Keypair = keypair
			secretOpts.Password = prd.git.Ssh.Password.ValueString()
		}
		secretOpts.SSHHostname = prd.git.Url.ValueURL().Host
	}
	return secretOpts, nil
}

func (prd *providerResourceData) CreateCommit(message string) (git.Commit, repository.CommitOption, error) {
	entityList, err := prd.GetEntityList()
	if err != nil {
		return git.Commit{}, nil, err
	}
	var signer *openpgp.Entity
	if entityList != nil {
		var err error
		signer, err = getOpenPgpEntity(entityList, prd.git.GpgPassphrase.ValueString(), prd.git.GpgKeyID.ValueString())
		if err != nil {
			return git.Commit{}, nil, fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	if prd.git.CommitMessageAppendix.ValueString() != "" {
		message = message + "\n\n" + prd.git.CommitMessageAppendix.ValueString()
	}
	commit := git.Commit{
		Author: git.Signature{
			Name:  prd.git.AuthorName.ValueString(),
			Email: prd.git.AuthorEmail.ValueString(),
		},
		Message: message,
	}
	return commit, repository.WithSigner(signer), nil
}

func (prd *providerResourceData) GetRepositoryURL() *url.URL {
	repositoryURL := prd.git.Url.ValueURL()
	if prd.git.Http != nil {
		repositoryURL.User = nil
	}
	if prd.git.Ssh != nil {
		if prd.git.Url.ValueURL().User == nil || prd.git.Ssh.Username.ValueString() != "git" {
			repositoryURL.User = url.User(prd.git.Ssh.Username.ValueString())
		}
	}
	return repositoryURL
}

func (prd *providerResourceData) GetEntityList() (openpgp.EntityList, error) {
	var entityList openpgp.EntityList
	if prd.git.GpgKeyRing.ValueString() != "" {
		var err error
		entityList, err = openpgp.ReadKeyRing(strings.NewReader(prd.git.GpgKeyRing.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("Failed to read GPG key ring: %w", err)
		}
	}
	return entityList, nil
}

func getOpenPgpEntity(keyRing openpgp.EntityList, passphrase, keyID string) (*openpgp.Entity, error) {
	if len(keyRing) == 0 {
		return nil, fmt.Errorf("empty GPG key ring")
	}

	var entity *openpgp.Entity
	if keyID != "" {
		if strings.HasPrefix(keyID, "0x") {
			keyID = strings.TrimPrefix(keyID, "0x")
		}
		if len(keyID) != 16 {
			return nil, fmt.Errorf("invalid GPG key id length; expected %d, got %d", 16, len(keyID))
		}
		keyID = strings.ToUpper(keyID)

		for _, ent := range keyRing {
			if ent.PrimaryKey.KeyIdString() == keyID {
				entity = ent
			}
		}

		if entity == nil {
			return nil, fmt.Errorf("no GPG keyring matching key id '%s' found", keyID)
		}
		if entity.PrivateKey == nil {
			return nil, fmt.Errorf("keyring does not contain private key for key id '%s'", keyID)
		}
	} else {
		entity = keyRing[0]
	}

	err := entity.PrivateKey.Decrypt([]byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt GPG private key: %w", err)
	}

	return entity, nil
}

func getAuthOpts(g *Git) (*git.AuthOptions, error) {
	u := g.Url.ValueURL()
	switch u.Scheme {
	case "http":
		if g.Http == nil {
			return nil, fmt.Errorf("Git URL scheme is http but http configuration is empty")
		}
		return &git.AuthOptions{
			Transport: git.HTTP,
			Username:  g.Http.Username.ValueString(),
			Password:  g.Http.Password.ValueString(),
		}, nil
	case "https":
		if g.Http == nil {
			return nil, fmt.Errorf("Git URL scheme is https but http configuration is empty")
		}
		return &git.AuthOptions{
			Transport: git.HTTPS,
			Username:  g.Http.Username.ValueString(),
			Password:  g.Http.Password.ValueString(),
			CAFile:    []byte(g.Http.CertificateAuthority.ValueString()),
		}, nil
	case "ssh":
		if g.Ssh == nil {
			return nil, fmt.Errorf("Git URL scheme is ssh but ssh configuration is empty")
		}
		if g.Ssh.PrivateKey.ValueString() != "" {
			kh, err := sourcesecret.ScanHostKey(u.Host)
			if err != nil {
				return nil, err
			}
			return &git.AuthOptions{
				Transport:  git.SSH,
				Username:   g.Ssh.Username.ValueString(),
				Password:   g.Ssh.Password.ValueString(),
				Identity:   []byte(g.Ssh.PrivateKey.ValueString()),
				KnownHosts: kh,
			}, nil
		}
		return nil, fmt.Errorf("ssh scheme cannot be used without private key")
	default:
		return nil, fmt.Errorf("scheme %q is not supported", u.Scheme)
	}
}

func getClientConfiguration(ctx context.Context, kubernetes *Kubernetes) (clientcmd.ClientConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if kubernetes.ConfigPath.ValueString() != "" {
		configPaths = []string{kubernetes.ConfigPath.ValueString()}
	} else if len(kubernetes.ConfigPaths.Elements()) > 0 {
		var pp []string
		diag := kubernetes.ConfigPaths.ElementsAs(ctx, &pp, false)
		if diag.HasError() {
			return nil, fmt.Errorf("%s", diag)
		}
		for _, p := range pp {
			configPaths = append(configPaths, p)
		}
	}
	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxSuffix := "; default context"
		if kubernetes.ConfigContext.ValueString() != "" || kubernetes.ConfigContextAuthInfo.ValueString() != "" || kubernetes.ConfigContextCluster.ValueString() != "" {
			ctxSuffix = "; overriden context"
			if kubernetes.ConfigContext.ValueString() != "" {
				overrides.CurrentContext = kubernetes.ConfigContext.ValueString()
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			}
			overrides.Context = clientcmdapi.Context{}
			if kubernetes.ConfigContextAuthInfo.ValueString() != "" {
				overrides.Context.AuthInfo = kubernetes.ConfigContextAuthInfo.ValueString()
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if kubernetes.ConfigContextCluster.ValueString() != "" {
				overrides.Context.Cluster = kubernetes.ConfigContextCluster.ValueString()
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = kubernetes.Insecure.ValueBool()
	if kubernetes.ClusterCACertificate.ValueString() != "" {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(kubernetes.ClusterCACertificate.ValueString()).Bytes()
	}
	if kubernetes.ClientCertificate.ValueString() != "" {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(kubernetes.ClientCertificate.ValueString()).Bytes()
	}
	if kubernetes.Host.ValueString() != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(kubernetes.Host.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	overrides.AuthInfo.Username = kubernetes.Username.ValueString()
	overrides.AuthInfo.Password = kubernetes.Password.ValueString()
	overrides.AuthInfo.Token = kubernetes.Token.ValueString()
	if kubernetes.ClientKey.ValueString() != "" {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(kubernetes.ClientKey.ValueString()).Bytes()
	}
	overrides.ClusterDefaults.ProxyURL = kubernetes.ProxyURL.ValueString()

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	return cc, nil
}
