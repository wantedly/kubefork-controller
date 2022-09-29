package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wantedly/kubefork-controller/kubeforkctl/domain"
	"github.com/wantedly/kubefork-controller/kubeforkctl/k8sClientGo"
)

type manifestOption struct {
	identifier           string
	namespace            string
	outputPath           string
	replicaNum           int32
	serviceLabel         []string
	serviceName          string
	deploymentLabel      []string
	deploymentName       string
	deploymentAnnotation map[string]string
	image                string
	env                  map[string]string
	forkManagerName      string
	validTime            int64
	kubeConfigPath       string
}

type manifestCmdRunner struct {
	option *manifestOption
}

func NewManifestCmd() *cobra.Command {
	opt := &manifestOption{}

	runner := manifestCmdRunner{
		option: opt,
	}

	cmd := &cobra.Command{
		Use:   "manifest",
		RunE:  runner.manifestRun,
		Short: "Generate manifest of Fork resource",
		Long: `Generate manifest of Fork resource based on option.
If you introduce kubefork-controller and deployment-duplicator, all you need is apply this to create Virtual Cluster.
When you apply a Fork resource, kubefork-controller create Mapping, DeploymentCopy, VirtualService, and forked Service resources.
Deployment-duplicator create forked Deployment from DeploymentCopy.
`,
	}

	cmd.Flags().StringVarP(&opt.identifier, "identifier", "i", "", "unique identifier for the forked resources")
	cmd.Flags().StringVarP(&opt.namespace, "namespace", "n", "", "target namespace")
	cmd.Flags().StringVarP(&opt.outputPath, "output", "o", "", "the path output fork manifest to file in the path specified by this option instead of standard output")
	cmd.Flags().Int32VarP(&opt.replicaNum, "replicas", "r", 1, "set replicas on deployment")
	cmd.Flags().StringArrayVar(&opt.serviceLabel, "service-label", []string{}, "label of service to be forked\n(support '<key>', '!<key>' '<key>=<value>' or '<key>!=<value>' formats)")
	cmd.Flags().StringVar(&opt.serviceName, "service-name", "", "name of service to be forked")
	cmd.Flags().StringArrayVar(&opt.deploymentLabel, "deployment-label", []string{}, "label of deployment to be forked\n(support '<key>', '!<key>' '<key>=<value>' or '<key>!=<value>' formats)")
	cmd.Flags().StringVar(&opt.deploymentName, "deployment-name", "", "name of deployment to be forked")
	cmd.Flags().StringToStringVar(&opt.deploymentAnnotation, "deployment-annotation", map[string]string{}, "annotation attached to forked deployment")
	cmd.Flags().StringVar(&opt.image, "image", "", "image of forked container\n(support '<name>:<tag>' format)")
	cmd.Flags().StringToStringVarP(&opt.env, "env", "e", map[string]string{}, "custom env vars for forked containers which ")
	cmd.Flags().StringVarP(&opt.forkManagerName, "fork-manager", "f", "", "name of fork manager\n(support '<namespace>/<name>' format)")
	cmd.Flags().Int64VarP(&opt.validTime, "valid-time", "v", 8, "valid time of fork resource (hour)")
	cmd.Flags().StringVarP(&opt.kubeConfigPath, "kubeconfig", "k", os.Getenv("KUBECONFIG"), "path of kubeconig\n(loading order follows the same rule as kubectl)")

	return cmd
}

func (m manifestCmdRunner) manifestRun(cmd *cobra.Command, _ []string) error {
	// Select client
	cli, err := k8sClientGo.NewClientSet(m.option.kubeConfigPath, cmd.Flags().Changed("kubeconfig"))
	if err != nil {
		return err
	}

	// Generate service selector
	serviceLabels := map[string]string{}
	if cmd.Flags().Changed("service-name") {
		serviceLabels, err = cli.GetAllServiceLabelsByServiceName(cmd.Context(), m.option.namespace, m.option.serviceName)
		if err != nil {
			return err
		}
	}

	serviceSelector, err := domain.NewSelector(m.option.serviceLabel, serviceLabels)
	if err != nil {
		return err
	}

	// Generate deployment selector
	deploymentLabels := map[string]string{}
	if cmd.Flags().Changed("deployment-name") {
		deploymentLabels, err = cli.GetAllDeploymentLabelsByDeploymentName(cmd.Context(), m.option.namespace, m.option.deploymentName)
		if err != nil {
			return err
		}
	}

	deploymentSelector, err := domain.NewSelector(m.option.deploymentLabel, deploymentLabels)
	if err != nil {
		return err
	}

	// The specified environment variables is overwritten in all containers where the image is switched
	env := domain.NewEnv(m.option.identifier, m.option.env)

	// Extract the containers whose images should be switched and create a manifest that changes the images in those containers
	containerNames, err := cli.GetAllContainersByImageNameAndServiceAndDeploymentSelector(cmd.Context(), m.option.namespace,
		m.option.image, serviceSelector, deploymentSelector)
	if err != nil {
		return err
	}
	containers := domain.NewContainers(m.option.image, containerNames, env)

	f := domain.NewFork(m.option.identifier, m.option.namespace, m.option.forkManagerName, m.option.replicaNum, time.Duration(m.option.validTime), serviceSelector, deploymentSelector,
		containers, m.option.deploymentAnnotation)

	if err := f.OutputManifest(m.option.outputPath); err != nil {
		return err
	}

	return nil
}
