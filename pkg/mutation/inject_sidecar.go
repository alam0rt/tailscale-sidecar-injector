package mutation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alam0rt/tailscale-sidecar-injector/pkg/headscale"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	SecretNameKey string = "TS_KUBE_SECRET"
	UserspaceKey  string = "TS_USERSPACE"
	PreAuthKeyKey string = "TS_AUTHKEY"
	TSExtraArgs   string = "TS_EXTRA_ARGS"
	// custom
	LoginServer string = "LOGIN_SERVER"
	APIKey      string = "API_KEY"
)

const (
	InjectLabel               string = "tailscale-inject"
	LoginServerAnnotation     string = "tailscale.iced.cool/login-server"
	SecretNameAnnotation      string = "tailscale.iced.cool/sercret-name"
	EnableUserspaceAnnotation string = "tailscale.iced.cool/userspace-enabled"
	// UserNameAnnotation defines which user to assume when creating pre-auth keys
	UserNameAnnotation string = "tailscale.iced.cool/user"
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
	Config config
}

type config struct {
	userspace   bool   // TS_USERSPACE
	preAuthKey  string // TS_AUTH_KEY
	secretName  string // TS_KUBE_SECRET
	loginServer string // TS_LOGIN_SERVER
	image       string
	user        string
	client      headscale.HeadscaleClient
}

func (c *config) LoginServer() string {
	return c.loginServer
}

var (
	ErrSecretNameNotProvided error = fmt.Errorf("%s missing: a secret containing the tailscale pre-auth-key must be provided", SecretNameKey)
	ErrSidecarNil            error = fmt.Errorf("provided sidecar was empty")
)

func getAnnotation(pod corev1.Pod, key string, defaultValue string) string {
	if v, ok := pod.Annotations[key]; ok {
		return v
	}
	return defaultValue
}

func (si sidecarInjector) buildConfig(pod corev1.Pod) (*config, error) {
	c := &config{}

	c.image = Image
	c.secretName = getAnnotation(pod, SecretNameAnnotation, defaultSecretName)
	c.userspace = getAnnotation(pod, EnableUserspaceAnnotation, "") != ""
	c.loginServer = getAnnotation(pod, LoginServerAnnotation, "")
	c.user = getAnnotation(pod, UserNameAnnotation, "")

	hs, err := headscale.New(context.TODO(), os.Getenv("HEADSCALE_CLI_API_KEY"), c.loginServer)
	if err != nil {
		return nil, err
	}
	c.client = hs

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
	pod.Spec.InitContainers = append([]corev1.Container{*sidecar}, pod.Spec.InitContainers...)
	return nil
}

func buildSidecarContainer(config *config) (*corev1.Container, error) {
	return &corev1.Container{
		Name:            "tailscale",
		Image:           config.image,
		ImagePullPolicy: corev1.PullAlways,
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
		Env: []corev1.EnvVar{
			{Name: SecretNameKey, Value: config.TSKubeSecret()},
			{Name: UserspaceKey, Value: config.TSUserspace()},
			{Name: TSExtraArgs, Value: strings.Join(config.TSExtraArgs(), " ")},
			// Todo: store in a secret
			{Name: PreAuthKeyKey, Value: config.preAuthKey},
		},
	}, nil

}

var _ podMutator = (*sidecarInjector)(nil)

func (si sidecarInjector) Name() string {
	return "sidecar_injector"
}

func (c *config) TSAuthKey(tags []string) (string, error) {
	if c.preAuthKey != "" {
		return c.preAuthKey, nil
	}

	var aclTags []string
	for _, tag := range tags {
		aclTags = append(aclTags, fmt.Sprintf("tag:%s", tag))
	}
	expiry := time.Now().Add(2 * time.Minute)
	resp, err := c.client.PreAuthKeys().Create(context.TODO(), c.user, false, true, expiry, aclTags)
	if err != nil {
		return "", err
	}
	c.preAuthKey = resp.PreAuthKey.Key
	return c.preAuthKey, nil

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

	if _, err := c.TSAuthKey([]string{
		pod.Namespace,
		"pod",
	}); err != nil {
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
