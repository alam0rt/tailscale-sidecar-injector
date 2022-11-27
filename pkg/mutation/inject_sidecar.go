package mutation

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// sidecarInjector implements the pod mutator interface
type sidecarInjector struct {
	Logger logrus.FieldLogger
}

type SidecarConfig struct {
	tsExtraArgs []string // TS_EXTRA_ARGS
	userspace   bool     // TS_USERSPACE
	preAuthKey  string   // TS_AUTH_KEY
	secretName  string   // TS_KUBE_SECRET
}

func buildSidecarConfig(pod *corev1.Pod) (*SidecarConfig, error) {
	loginServer := loginServer(pod)
	c := &SidecarConfig{}

	// configure with login server if annotation is present
	if loginServer != "" {
		c.tsExtraArgs = append(c.tsExtraArgs, loginServer)
	}

}

const (
	SecretNameKey string = "TS_KUBE_SECRET"
	UserspaceKey  string = "TS_USERSPACE"
	PreAuthKeyKey string = "TS_AUTH_KEY"
	TSExtraArgs   string = "TS_EXTRA_ARGS"
)

const (
	LoginServerAnnotation string = "samlockart.io/login-server"
)

// TODO: provide via flags / config
const (
	SecretName string = "tailscale-auth"
)

func loginServer(pod *corev1.Pod) string {
	const flag string = "--login-server=%s"

	if pod == nil {
		return ""
	}

	if v, ok := pod.Annotations[LoginServerAnnotation]; ok {
		return fmt.Sprintf(flag, v)

	}
	return ""
}

func injectSidecar(pod *corev1.Pod) error {
	current := pod.Spec.DeepCopy().Containers
	pod.Spec.Containers = nil
	pod.Spec.Containers = []corev1.Container{}
}

func buildSidecarContainer(config *SidecarConfig) (*corev1.Container, error) {
	return &corev1.Container{
		Name:            "tailscale-sidecar",
		Image:           "ghcr.io/tailscale/tailscale:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
		Env: []corev1.EnvVar{
			{Name: SecretNameKey, Value: SecretName},
			{Name: UserspaceKey, Value: "false"},
			{Name: TSExtraArgs, Value: strings.Join([]string{loginServer}, " ")},
			{
				Name: PreAuthKeyKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key:                  "TS_AUTH_KEY",
						LocalObjectReference: corev1.LocalObjectReference{Name: SecretName},
						Optional:             &[]bool{false}[0],
					},
				},
			},
		},
	}, nil

}

var _ podMutator = (*sidecarInjector)(nil)

func (si sidecarInjector) Name() string {
	return "sidecar_injector"
}

func (si sidecarInjector) Mutate(pod *corev1.Pod) (*corev1.Pod, error) {
	si.Logger = si.Logger.WithField("mutation", si.Name())
	mpod := pod.DeepCopy()

	return nil, nil
}
