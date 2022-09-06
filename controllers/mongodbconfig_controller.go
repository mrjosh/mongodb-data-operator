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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	"github.com/mrjosh/mongodb-data-operator/pkg/mongodb"
	"github.com/pingcap/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MongoDBConfigReconciler reconciles a MongoDBConfig object
type MongoDBConfigReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mongo.snappcloud.io,resources=mongodbconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

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
			return doNotRequeue()
		}

		log.Error(err, "failed to get the mongo-config")
		return requeue(err)
	}

	metaObj, err := meta.Accessor(mongoCfg)
	if err != nil {
		log.Error(err, "unable to get metadata for object")
		return reconcile.Result{}, err
	}

	// ignore resources that are being deleted
	if !metaObj.GetDeletionTimestamp().IsZero() {
		log.Info("ignoring", "reason", "object has a non-zero deletion timestamp")
		return reconcile.Result{}, nil
	}

	if mongoCfg.Spec.MongoURL == "" {

		r.Recorder.Event(
			mongoCfg,
			corev1.EventTypeWarning,
			string(mongov1.NoMongoURLSpecified),
			"Specifying an MongoURL for the MongoDBConfig is required",
		)

		mongoCfg.Status.Ready = string(corev1.ConditionFalse)
		apimeta.SetStatusCondition(&mongoCfg.Status.Conditions, metav1.Condition{
			Type:               string(mongov1.NoMongoURLSpecified),
			Status:             metav1.ConditionFalse,
			Reason:             string(mongov1.NoMongoURLSpecified),
			Message:            "Specifying an MongoURL for the MongoDBConfig is required",
			ObservedGeneration: mongoCfg.Generation,
		})

		if err := r.Client.Status().Update(ctx, mongoCfg); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return doNotRequeue()
	}

	// try to connect to the mongodb url
	// create a new mongodb client for connection validation
	if _, err := mongodb.NewClientWithContext(ctx, mongoCfg.Spec.MongoURL); err != nil {

		message := err.Error()

		r.Recorder.Event(
			mongoCfg,
			corev1.EventTypeWarning,
			string(mongov1.ConnectError),
			message,
		)

		mongoCfg.Status.Ready = string(corev1.ConditionFalse)
		apimeta.SetStatusCondition(&mongoCfg.Status.Conditions, metav1.Condition{
			Type:               string(mongov1.ConnectError),
			Status:             metav1.ConditionFalse,
			Reason:             string(mongov1.ConnectError),
			Message:            message,
			ObservedGeneration: mongoCfg.Generation,
		})

		if err := r.Client.Status().Update(ctx, mongoCfg); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return requeueWithDelay(30 * time.Second)
	}

	readyMessage := "Successfully connected to MongoDB database"

	r.Recorder.Event(
		mongoCfg,
		corev1.EventTypeNormal,
		string(mongov1.Ready),
		readyMessage,
	)

	mongoCfg.Status.Ready = string(corev1.ConditionTrue)
	apimeta.SetStatusCondition(&mongoCfg.Status.Conditions, metav1.Condition{
		Type:               string(mongov1.Ready),
		Status:             metav1.ConditionTrue,
		Reason:             string(mongov1.Ready),
		Message:            readyMessage,
		ObservedGeneration: mongoCfg.Generation,
	})

	if err := r.Client.Status().Update(ctx, mongoCfg); err != nil {
		log.Error(err, "unable to update target's status object")
		return requeue(err)
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
