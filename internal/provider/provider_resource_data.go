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

	repositoryUrl         *url.URL
	branch                string
	authorName            string
	authorEmail           string
	commitMessageAppendix string
	entityList            openpgp.EntityList
	gpgPassphrase         string
	gpgID                 string
	secretOpts            sourcesecret.Options
	authOpts              *git.AuthOptions
	clientOpts            []gogit.ClientOption
}

func NewProviderResourceData(ctx context.Context, data ProviderModel) (*providerResourceData, error) {
	// Kubernetes configuration
	clientCfg, err := getClientConfiguration(ctx, data.Kubernetes)
	if err != nil {
		return nil, fmt.Errorf("Invalid Kubernetes configuration: %w", err)
	}
	rcg := utils.NewRestClientGetter(clientCfg)

	// Git configuration
	authOpts, err := getAuthOpts(data.Git)
	if err != nil {
		return nil, err
	}
	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage()}
	if data.Git.Http != nil && data.Git.Http.InsecureHttpAllowed.ValueBool() {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
	}
	var entityList openpgp.EntityList
	if data.Git.GpgKeyRing.ValueString() != "" {
		var err error
		entityList, err = openpgp.ReadKeyRing(strings.NewReader(data.Git.GpgKeyRing.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("Failed to read GPG key ring: %w", err)
		}
	}
	secretOpts := sourcesecret.Options{}
	if data.Git.Http != nil {
		secretOpts.Username = data.Git.Http.Username.ValueString()
		secretOpts.Password = data.Git.Http.Password.ValueString()
		secretOpts.CAFile = []byte(data.Git.Http.CertificateAuthority.ValueString())
	}
	if data.Git.Ssh != nil {
		if data.Git.Ssh.PrivateKey.ValueString() != "" {
			keypair, err := sourcesecret.LoadKeyPair([]byte(data.Git.Ssh.PrivateKey.ValueString()), data.Git.Ssh.Password.ValueString())
			if err != nil {
				return nil, fmt.Errorf("Failed to load SSH Key Pair: %w", err)
			}
			secretOpts.Keypair = keypair
			secretOpts.Password = data.Git.Ssh.Password.ValueString()
		}
		secretOpts.SSHHostname = data.Git.Url.ValueURL().Host
	}

	return &providerResourceData{
		rcg:                   rcg,
		repositoryUrl:         data.Git.Url.ValueURL(),
		branch:                data.Git.Branch.ValueString(),
		authorName:            data.Git.AuthorName.ValueString(),
		authorEmail:           data.Git.AuthorEmail.ValueString(),
		commitMessageAppendix: data.Git.CommitMessageAppendix.ValueString(),
		entityList:            entityList,
		gpgPassphrase:         data.Git.GpgPassphrase.ValueString(),
		gpgID:                 data.Git.GpgKeyID.ValueString(),
		secretOpts:            secretOpts,
		authOpts:              authOpts,
		clientOpts:            clientOpts,
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
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary working directory for git repository: %w", err)
	}
	client, err := gogit.NewClient(tmpDir, prd.authOpts, prd.clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create git client: %w", err)
	}
	// TODO: Need to conditionally clone here. If repository is empty this will fail.
	_, err = client.Clone(ctx, prd.repositoryUrl.String(), repository.CloneOptions{CheckoutStrategy: repository.CheckoutStrategy{Branch: prd.branch}})
	if err != nil {
		return nil, fmt.Errorf("could not clone git repository: %w", err)
	}
	return client, nil
}

func (prd *providerResourceData) GetBootstrapOptions() ([]bootstrap.GitOption, error) {
	return []bootstrap.GitOption{
		bootstrap.WithRepositoryURL(prd.repositoryUrl.String()),
		bootstrap.WithKubeconfig(prd.rcg, &runclient.Options{}),
		bootstrap.WithBranch(prd.branch),
		bootstrap.WithSignature(prd.authorName, prd.authorEmail),
		bootstrap.WithCommitMessageAppendix(prd.commitMessageAppendix),
		bootstrap.WithGitCommitSigning(prd.entityList, prd.gpgPassphrase, prd.gpgID),
		bootstrap.WithLogger(log.NopLogger{}),
	}, nil
}

func (prd *providerResourceData) CreateCommit(message string) (git.Commit, repository.CommitOption, error) {
	var signer *openpgp.Entity
	if prd.entityList != nil {
		var err error
		signer, err = getOpenPgpEntity(prd.entityList, prd.gpgPassphrase, prd.gpgID)
		if err != nil {
			return git.Commit{}, nil, fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	if prd.commitMessageAppendix != "" {
		message = message + "\n\n" + prd.commitMessageAppendix
	}
	commit := git.Commit{
		Author: git.Signature{
			Name:  prd.authorName,
			Email: prd.authorEmail,
		},
		Message: message,
	}
	return commit, repository.WithSigner(signer), nil
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

	// Validate that the Kubernetes configuration is correct.
	if _, err := cc.ClientConfig(); err != nil {
		return nil, err
	}

	return cc, nil
}
