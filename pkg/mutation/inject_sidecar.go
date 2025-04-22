package mutation

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	SecretNameKey string = "TS_KUBE_SECRET"
	UserspaceKey  string = "TS_USERSPACE"
	PreAuthKeyKey string = "TS_AUTH_KEY"
	TSExtraArgs   string = "TS_EXTRA_ARGS"
	// custom
	LoginServer string = "TS_LOGIN_SERVER"
)

const (
	InjectLabel               string = "tailscale-inject"
	LoginServerAnnotation     string = "iced.cool/tailscale/login-server"
	SecretNameAnnotation      string = "iced.cool/tailscale/sercret-name"
	EnableUserspaceAnnotation string = "iced.cool/tailscale/enable-userspace"
)

const (
	ImageName         = "ghcr.io/tailscale/tailscale"
	ImageTag          = "latest"
	Image             = ImageName + ":" + ImageTag
	defaultSecretName = "tailscale-auth"
)

// sidecarInjector implements the pod mutator interface
type sidecarInjector struct {
	Logger logrus.FieldLogger
}

type config struct {
	userspace   bool   // TS_USERSPACE
	preAuthKey  string // TS_AUTH_KEY
	secretName  string // TS_KUBE_SECRET
	loginServer string // TS_LOGIN_SERVER
	image       string
}

func (c *config) LoginServer() string {
	return c.loginServer
}

var (
	ErrSecretNameNotProvided error = fmt.Errorf("%s missing: a secret containing the tailscale pre-auth-key must be provided", SecretNameKey)
	ErrSidecarNil            error = fmt.Errorf("provided sidecar was empty")
)

func (si sidecarInjector) buildConfig(pod corev1.Pod) (*config, error) {
	c := &config{}

	// TODO: user definable
	c.image = Image

	// get the name of the secret containing the pre-auth-key
	if v, ok := pod.Annotations[SecretNameAnnotation]; ok {
		c.secretName = v
	} else {
		c.secretName = defaultSecretName
	}

	// enable or disable userspace mode
	if _, ok := pod.Annotations[EnableUserspaceAnnotation]; ok {
		c.userspace = true
	}

	// configure with login server if annotation is present
	if v, ok := pod.Annotations[LoginServerAnnotation]; ok {
		c.loginServer = v
	}

	return c, nil
}

func (c *config) TSExtraArgs() []string {
	var args []string

	if c.loginServer != "" {
		args = append(args, fmt.Sprintf("--login-server=%s", c.loginServer))
	}

	return args
}

func (c *config) TSKubeSecret() string {
	return c.secretName
}

func (c *config) TSAuthKey() string {
	return c.preAuthKey
}

func (c *config) TSUserspace() string {
	if c.userspace {
		return "true"
	}
	return "false"
}

func injectSidecar(pod *corev1.Pod, sidecar *corev1.Container) error {
	if sidecar == nil {
		return ErrSidecarNil
	}
	pod.Spec.InitContainers = append([]corev1.Container{*sidecar}, pod.Spec.Containers...)
	return nil
}

func buildSidecarContainer(config *config) (*corev1.Container, error) {
	return &corev1.Container{
		Name:            "tailscale",
		Image:           config.image,
		ImagePullPolicy: corev1.PullAlways,
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
		Env: []corev1.EnvVar{
			{Name: SecretNameKey, Value: config.TSKubeSecret()},
			{Name: UserspaceKey, Value: config.TSUserspace()},
			{Name: TSExtraArgs, Value: strings.Join(config.TSExtraArgs(), " ")},
			{
				Name: PreAuthKeyKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key:                  PreAuthKeyKey,
						LocalObjectReference: corev1.LocalObjectReference{Name: config.TSKubeSecret()},
						Optional:             ptr.To(false),
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
	// build the logger
	si.Logger = si.Logger.WithField("mutation", si.Name())

	if _, ok := pod.Labels[InjectLabel]; !ok {
		si.Logger.Infof("ignoring %s", pod.Name)
		return pod, nil
	}

	c, err := si.buildConfig(*pod)
	if err != nil {
		return nil, err
	}

	sc, err := buildSidecarContainer(c)
	if err != nil {
		return nil, err
	}

	// inject the sidecar
	mpod := pod.DeepCopy()
	injectSidecar(mpod, sc)

	return mpod, nil
}
