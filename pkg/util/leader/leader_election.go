package leader

import (
	"context"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	leaderelection "k8s.io/client-go/tools/leaderelection"
	leaderelectionresourcelock "k8s.io/client-go/tools/leaderelection/resourcelock"
)

// RunLeaderElection runs leader election using Kubernetes LeaseLock
func RunLeaderElection(
	ctx context.Context,
	kubeClient *kubernetes.Clientset,
	identity string,
	lockName string,
	namespace string,
	onStartedLeading func(ctx context.Context) error,
	onStoppedLeading func(),
) error {
	if namespace == "" {
		namespace = "default"
	}
	lock := &leaderelectionresourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      lockName,
			Namespace: namespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: leaderelectionresourcelock.ResourceLockConfig{
			Identity: identity,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				if err := onStartedLeading(ctx); err != nil {
					log.Printf("[LeaderElection] Error in OnStartedLeading: %v", err)
					return
				}
			},
			OnStoppedLeading: func() {
				log.Println("[LeaderElection] Lost leadership, shutting down.")
				if onStoppedLeading != nil {
					onStoppedLeading()
				}
			},
			OnNewLeader: func(newIdentity string) {
				if newIdentity == identity {
					log.Println("[LeaderElection] I am the new leader.")
				} else {
					log.Printf("[LeaderElection] New leader elected: %s", newIdentity)
				}
			},
		},
		ReleaseOnCancel: true,
	})

	return nil
}
