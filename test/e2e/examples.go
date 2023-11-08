package e2e

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestXonoticGameServerReady(t *testing.T) {
	t.Parallel()

	// Create a xonotic GameServer
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "xonotic-",
			Namespace:    viper.GetString("gameserversNamespace"),
		},
		Spec: agonesv1.GameServerSpec{
			Container: "xonotic",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 26000,
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "xonotic",
							Image: "us-docker.pkg.dev/agones-images/examples/xonotic-example:1.4",
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, gs.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	// Assert that the GameServer is in the expected state
	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State, "GameServer in Ready state")
}
