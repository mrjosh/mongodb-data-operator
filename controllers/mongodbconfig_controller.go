/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	"github.com/mrjosh/mongodb-data-operator/pkg/mongodb"
	"github.com/pingcap/errors"
)

// MongoDBConfigReconciler reconciles a MongoDBConfig object
type MongoDBConfigReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MongoDBConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MongoDBConfigReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	log := r.Log.WithValues("mongodb-config", req.NamespacedName)
	log.Info("Reconciling MongoDBConfig")

	mongoCfg := &mongov1.MongoDBConfig{}
	if err := r.Get(ctx, req.NamespacedName, mongoCfg); err != nil {

		if errors.IsNotFound(err) {
			// don't requeue on deletions, which yield a non-found object
			log.Info("ignoring", "reason", "not found", "err", err)
		}

		log.Error(err, "failed to get the mongo-config")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// try to connect to the mongodb url
	// create a new mongodb client for connection validation
	if _, err := mongodb.NewClientWithContext(ctx, mongoCfg.Spec.MongoURL); err != nil {

		if err := r.setEventStatusError(ctx, mongoCfg, err.Error()); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return requeueWithDelay(30 * time.Second)
	}

	if mongoCfg.Status.Ready != string(mongov1.Ready) {
		if err := r.setEventStatusReady(ctx, mongoCfg, "Successfully connected to MongoDB database"); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}
	}

	return doNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mongov1.MongoDBConfig{}).
		Complete(r)
}

func doNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func requeue(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func requeueWithDelay(td time.Duration) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: td}, nil
}
