// Copyright 2024 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// This custom controller is deployed in the Kubernetes cluster, specifically in the agones-system namespace,
// and it's designed to observe and react to changes in GameServer instances.

package main

import (
	"context"
	"fmt"
	"os"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// GameServerReconciler reconciles a GameServer object
type GameServerReconciler struct {
	client.Client
	Log logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("gameserver", req.NamespacedName)

	// Fetch the GameServer instance
	gameServer := &agonesv1.GameServer{}

	log.Info("Fetching GameServer", "gameServerName", gameServer.Name, "gameServerNamespace", gameServer.Namespace)

	err := r.Get(ctx, req.NamespacedName, gameServer)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("GameServer resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch GameServer")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// logging the GameServer's name and current state
	log.Info("GameServer event observed", "name", gameServer.Name, "state", gameServer.Status.State)

	if gameServer.Labels == nil {
		gameServer.Labels = map[string]string{}
	}
	gameServer.Labels["state"] = fmt.Sprintf("%v", gameServer.Status.State)

	log.Info("GameServer event observed", "name", gameServer.Name, "state", gameServer.Status.State)

	// Attempt to update the GameServer with a retry loop for handling conflicts
	retryAttempts := 5
	for i := 0; i < retryAttempts; i++ {
		log.Info("Entering into retry loop", "attempt", i)
		err = r.Update(ctx, gameServer)
		if err != nil {
			if i < retryAttempts-1 && errors.IsConflict(err) {
				// Conflict detected, refetch the latest version and retry
				_ = r.Get(ctx, req.NamespacedName, gameServer)
				continue
			}
			log.Error(err, "failed to update GameServer")
			return ctrl.Result{}, err
		}
		// Update succeeded
		break
	}

	log.Info("GameServer updated successfully", "name", gameServer.Name)

	return ctrl.Result{}, nil
}

func main() {
	var err error
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		fmt.Println("Unable to start manager", err)
		os.Exit(1)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&agonesv1.GameServer{}).
		Complete(&GameServerReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("GameServer"),
		})
	if err != nil {
		fmt.Println("Unable to create controller", err)
		os.Exit(1)
	}

	// Start the controller manager
	fmt.Println("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Println("Problem running manager", err)
		os.Exit(1)
	}
}
