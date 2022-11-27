package mutation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectSidecarMutate(t *testing.T) {
	want := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				SecretNameAnnotation: "foo",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "test",
				Env: []corev1.EnvVar{
					{
						Name:  "KUBE",
						Value: "true",
					},
				},
			}},
			InitContainers: []corev1.Container{{
				Name: "inittest",
				Env: []corev1.EnvVar{
					{
						Name:  "KUBE",
						Value: "true",
					},
				},
			}},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				SecretNameAnnotation: "foo",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "test",
			}},
			InitContainers: []corev1.Container{{
				Name: "inittest",
			}},
		},
	}

	got, err := sidecarInjector{Logger: logger()}.Mutate(pod)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, got)
}
