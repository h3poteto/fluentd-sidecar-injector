// https://github.com/kubernetes/client-go/blob/master/examples/leader-election/main.go
package leaderelection

import (
	"context"
	"os"
	"time"

	"github.com/h3poteto/fluentd-sidecar-injector/pkg/signals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

type LeaderElection struct {
	name      string
	namespace string
}

func NewLeaderElection(name, namespace string) *LeaderElection {
	return &LeaderElection{
		name,
		namespace,
	}
}

func (le *LeaderElection) Run(ctx context.Context, cfg *rest.Config, run func(ctx context.Context, clientConfig *rest.Config, stopCh <-chan struct{})) error {
	stopCh := signals.SetupSignalHandler()

	client := clientset.NewForConfigOrDie(cfg)

	id := string(uuid.NewUUID())
	klog.Infof("leader election id: %s", id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      le.name,
			Namespace: le.namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				defer cancel()
				run(ctx, cfg, stopCh)
				os.Exit(0)
			},
			OnStoppedLeading: func() {
				klog.Infof("leader lost: %s", id)
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					// I just got the lock
					return
				}
				klog.Infof("new leader elected: %s", identity)
			},
		},
	})
	return nil
}
