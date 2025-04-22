package mutation

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIgnoreUnannotatedPod(t *testing.T) {
	want := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "test",
			}},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "test",
			}},
		},
	}

	logger := logrus.New().WithField("test", t.Name())
	got, err := sidecarInjector{Logger: logger}.Mutate(pod)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, want, got)
}

func TestInjectSidecarMutate(t *testing.T) {
	tests := map[string]struct {
		got  *corev1.Pod
		want *corev1.Pod
	}{
		"doesnt inject": {
			&corev1.Pod{},
			&corev1.Pod{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			logger := logrus.New()
			logger.WithField("test", t.Name())
			want := test.want.DeepCopy()
			got, err := sidecarInjector{Logger: logger}.Mutate(test.got.DeepCopy())
			assert.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
