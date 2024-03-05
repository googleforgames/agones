// Copyright 2024 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// Setup the logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	log := ctrl.Log.WithName("controller-example")

	// Initialize the controller manager
	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	// Set up the GameServer controller with the manager
	err = ctrl.NewControllerManagedBy(manager).
		For(&agonesv1.GameServer{}). // Watch for GameServer events
		Complete(&GameServerReconciler{
			Client: manager.GetClient(),
		})
	if err != nil {
		log.Error(err, "could not create GameServer controller")
		os.Exit(1)
	}

	// Start the controller manager
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}

// GameServerReconciler is the controller implementation for GameServer resources
type GameServerReconciler struct {
	client.Client
	Log logr.Logger
}

// Reconcile responds to changes in GameServer resources
func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("reconcile").WithValues("gameserver", req.NamespacedName)

	var gameServer agonesv1.GameServer
	if err := r.Get(ctx, req.NamespacedName, &gameServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// logging the GameServer's name and current state
	log.Info("GameServer event observed", "name", gameServer.Name, "state", gameServer.Status.State)

	// Update a label on the GameServer with its current state
	if gameServer.Labels == nil {
		gameServer.Labels = make(map[string]string)
	}
	gameServer.Labels["state"] = fmt.Sprintf("%v", gameServer.Status.State)
	if err := r.Update(ctx, &gameServer); err != nil {
		log.Error(err, "failed to update GameServer labels")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
