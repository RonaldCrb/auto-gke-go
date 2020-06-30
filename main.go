package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v3/go/gcp/container"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/providers"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		clusterName := "demo-cluster"

		appLabels := pulumi.StringMap{
			"app": pulumi.String("pickard-app"),
		}

		version, err := findLatestGKEversion(ctx)
		if err != nil {
			return err
		}

		cluster, err := createCluster(ctx, clusterName, version)
		if err != nil {
			return err
		}

		provider, err := createProvider(ctx, cluster)
		if err != nil {
			return err
		}

		namespace, err := createNamespace(ctx, provider)
		if err != nil {
			return err
		}

		deploy, err := createDeploy(ctx, "pickard-demo", provider, namespace, appLabels)
		if err != nil {
			return err
		}

		service, err := createService(ctx, "pickard-service", provider, namespace, appLabels)
		if err != nil {
			return err
		}

		fmt.Printf("deployment URN => %s", deploy.URN)
		fmt.Printf("service URN => %s", service.URN)

		ctx.Export(
			"kubeconfig",
			generateKubeconfig(
				cluster.Endpoint,
				cluster.Name,
				cluster.MasterAuth,
			),
		)
		return nil
	})

}

func findLatestGKEversion(ctx *pulumi.Context) (string, error) {
	// get available GKE versions
	engineVersions, err := container.GetEngineVersions(
		ctx,
		&container.GetEngineVersionsArgs{},
	)
	// get the latest GKE versions
	version := engineVersions.LatestMasterVersion

	return version, err
}

// returns a cluster or an error
func createCluster(ctx *pulumi.Context, name string, v string) (*container.Cluster, error) {

	return container.NewCluster(ctx, "demo-cluster", &container.ClusterArgs{
		InitialNodeCount: pulumi.Int(2),
		MinMasterVersion: pulumi.String(v),
		NodeVersion:      pulumi.String(v),
		NodeConfig: &container.ClusterNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-1"),
			OauthScopes: pulumi.StringArray{
				pulumi.String("https://www.googleapis.com/auth/compute"),
				pulumi.String("https://www.googleapis.com/auth/devstorage.read_only"),
				pulumi.String("https://www.googleapis.com/auth/logging.write"),
				pulumi.String("https://www.googleapis.com/auth/monitoring"),
			},
		},
	})
}

// confection a kubeconfig file
func generateKubeconfig(
	clusterEndpoint pulumi.StringOutput,
	clusterName pulumi.StringOutput,
	clusterMasterAuth container.ClusterMasterAuthOutput,
) pulumi.StringOutput {
	context := pulumi.Sprintf("demo_%s", clusterName)

	return pulumi.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: https://%s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
    auth-provider:
      config:
        cmd-args: config config-helper --format=json
        cmd-path: gcloud
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp`,
		clusterMasterAuth.ClusterCaCertificate().Elem(),
		clusterEndpoint, context, context, context, context, context, context)
}

// returns a cluster provider for other resources to depend on
func createProvider(ctx *pulumi.Context, cluster *container.Cluster) (*providers.Provider, error) {
	return providers.NewProvider(
		ctx,
		"demo-provider",
		&providers.ProviderArgs{
			Kubeconfig: generateKubeconfig(
				cluster.Endpoint,
				cluster.Name,
				cluster.MasterAuth,
			),
		},
		pulumi.DependsOn([]pulumi.Resource{cluster}),
	)
}

// returns a new namespace
func createNamespace(ctx *pulumi.Context, provider *providers.Provider) (*corev1.Namespace, error) {
	return corev1.NewNamespace(
		ctx,
		"app-ns",
		&corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("demo-ns"),
			},
		},
		pulumi.Provider(provider))
}

func createDeploy(
	ctx *pulumi.Context,
	name string,
	p *providers.Provider,
	n *corev1.Namespace,
	l pulumi.StringMap,
) (*appsv1.Deployment, error) {

	return appsv1.NewDeployment(
		ctx,
		name,
		&appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: n.Metadata.Elem().Name(),
			},
			Spec: appsv1.DeploymentSpecArgs{
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: l,
				},
				Replicas: pulumi.Int(3),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: l,
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Name:  pulumi.String(name),
								Image: pulumi.String("ronaldcrb/node-pickard"),
							}},
					},
				},
			},
		}, pulumi.Provider(p))
}

func createService(
	ctx *pulumi.Context,
	name string,
	p *providers.Provider,
	n *corev1.Namespace,
	l pulumi.StringMap,
) (*corev1.Service, error) {
	return corev1.NewService(
		ctx,
		name,
		&corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: n.Metadata.Elem().Name(),
				Labels:    l,
			},
			Spec: &corev1.ServiceSpecArgs{
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(3000),
					},
				},
				Selector: l,
				Type:     pulumi.String("LoadBalancer"),
			},
		}, pulumi.Provider(p))
}
