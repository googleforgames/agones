package leader

import (
	"context"
	"fmt"
	"time"

	"agones.dev/agones/pkg/util/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RunLeaderTracking watches the specified leader lease and calls onLeaderChange with the new leader address.
// onLeaderChange receives a context (cancelled when the leader changes), the leader address.
func RunLeaderTracking(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	leaseName string,
	namespace string,
	port int,
	onLeaderChange func(clientCtx context.Context, addr string),
) {
	var (
		cancel     context.CancelFunc
		lastLeader string
		logger     = runtime.NewLoggerWithSource("RunLeaderTracking")
	)

	logger.Infof("Starting RunLeaderTracking for lease %q in namespace %q", leaseName, namespace)
	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled, stopping RunLeaderTracking")
			if cancel != nil {
				cancel()
			}
			return
		default:
		}
		lease, err := kubeClient.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
		if err != nil {
			logger.WithError(err).Warnf("Failed to get Lease %q in namespace %q", leaseName, namespace)
		} else if lease.Spec.HolderIdentity != nil {
			leader := *lease.Spec.HolderIdentity
			if leader != lastLeader {
				if cancel != nil {
					cancel()
				}
				pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, leader, metav1.GetOptions{})
				if err != nil {
					logger.WithError(err).Errorf("Failed to get Pod %q in namespace %q", leader, namespace)
				} else {
					leaderPodIP := pod.Status.PodIP
					addr := fmt.Sprintf("%s:%d", leaderPodIP, port)
					clientCtx, cancelFunc := context.WithCancel(ctx)
					cancel = cancelFunc
					runtime.NewLoggerWithSource("leadertracker").WithField("processorAddr", addr).Info("Connecting to new processor leader (using Pod IP)")
					go onLeaderChange(clientCtx, addr)
				}
				lastLeader = leader
			}
		} else {
			logger.Warnf("Lease %q in namespace %q has no HolderIdentity", leaseName, namespace)
		}
		time.Sleep(10 * time.Second)
	}
}
