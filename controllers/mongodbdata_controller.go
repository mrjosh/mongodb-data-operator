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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	"github.com/pingcap/errors"
)

// MongoDBDataReconciler reconciles a MongoDBData object
type MongoDBDataReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbdata,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbdata/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbdata/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MongoDBData object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MongoDBDataReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("mongodb-data", req.NamespacedName)

	log.Info("Reconciling MongoDBData")

	mongoData := &mongov1.MongoDBData{}
	if err := r.Get(ctx, req.NamespacedName, mongoData); err != nil {
		if errors.IsNotFound(err) {
			// don't requeue on deletions, which yield a non-found object
			log.Info("ignoring", "reason", "not found", "err", err)
			return doNotRequeue()
		}

		log.Error(err, "failed to get the mongo-data")
		return requeue(err)
	}

	metaObj, err := meta.Accessor(mongoData)
	if err != nil {
		log.Error(err, "unable to get metadata for object")
		return requeue(err)
	}

	// ignore resources that are being deleted
	if !metaObj.GetDeletionTimestamp().IsZero() {
		log.Info("ignoring", "reason", "object has a non-zero deletion timestamp")
		return doNotRequeue()
	}

	// Check if the MongoDBConfig exists
	mongoCfg := &mongov1.MongoDBConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: mongoData.Spec.DB}, mongoCfg); err != nil {

		if errors.IsNotFound(err) {

			message := fmt.Sprintf("MongoDBConfig with name %s doesn't exists", mongoData.Spec.DB)

			err := r.setEventStatusCondition(
				ctx,
				mongoData,
				mongov1.MongoDBDataConditionPending,
				metav1.ConditionFalse,
				message,
			)

			if err != nil {
				log.Error(err, "unable to update target's status object")
				return requeue(err)
			}

			log.Info("ignoring", "reason", message)
			return requeueWithDelay(20 * time.Second)
		}

		log.Error(err, fmt.Sprintf("failed to get the mongo-config %s", mongoData.Spec.DB))
		return requeue(err)
	}

	// Insert data to MongoDBConfig collection

	err = r.setEventStatusCondition(
		ctx,
		mongoData,
		mongov1.MongoDBDataConditionInserted,
		metav1.ConditionTrue,
		fmt.Sprintf("MongoDBData successfully Inserted into %s collection", mongoCfg.Spec.Collection),
	)

	if err != nil {
		log.Error(err, "unable to update target's status object")
		return requeue(err)
	}

	return doNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDataReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mongov1.MongoDBData{}).
		Complete(r)
}
