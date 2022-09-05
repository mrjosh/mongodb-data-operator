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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	"github.com/pingcap/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MongoDBConfigReconciler reconciles a MongoDBConfig object
type MongoDBConfigReconciler struct {
	client.Client
	Log logr.Logger

	Scheme *runtime.Scheme
}

var (
	reconcilePeriod        = 30 * time.Second
	reconcileResultRequeue = reconcile.Result{RequeueAfter: reconcilePeriod}
)

//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/finalizers,verbs=update

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

	mongoCfg := &mongov1.MongoDBConfig{}
	key := types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}

	if err := r.Get(ctx, key, mongoCfg); err != nil {
		if errors.IsNotFound(err) {
			// don't requeue on deletions, which yield a non-found object
			r.Log.Info("ignoring", "reason", "not found", "err", err)
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "unable to fetch target object to inject into")
		return reconcileResultRequeue, nil
	}

	metaObj, err := meta.Accessor(mongoCfg)
	if err != nil {
		r.Log.Error(err, "unable to get metadata for object")
		return reconcile.Result{}, err
	}

	// ignore resources that are being deleted
	if !metaObj.GetDeletionTimestamp().IsZero() {
		r.Log.Info("ignoring", "reason", "object has a non-zero deletion timestamp")
		return reconcile.Result{}, nil
	}

	if mongoCfg.Spec.MongoURL == "" {
		if mongoCfg.Status.Conditions == nil {
			mongoCfg.Status.Conditions = []mongov1.MongoDBConfigCondition{}
		}
		mongoCfg.Status.Ready = v1.ConditionFalse
		mongoCfg.Status.Status = mongov1.NoMongoURLSpecified
		mongoCfg.Status.Conditions = append(mongoCfg.Status.Conditions, mongov1.MongoDBConfigCondition{
			Type:               mongoCfg.Status.Status,
			Status:             mongoCfg.Status.Ready,
			LastTransitionTime: metav1.Now(),
			Reason:             string(mongov1.NoMongoURLSpecified),
			Message:            "Specifying an MongoURL for the MongoDBConfig is required",
		})

		if err := r.Client.Status().Update(ctx, mongoCfg); err != nil {
			r.Log.Error(err, "unable to update target's status object")
			return reconcile.Result{}, err
		}
	}

	// try to connect to the mongodb url for health check here

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mongov1.MongoDBConfig{}).
		Complete(r)
}
