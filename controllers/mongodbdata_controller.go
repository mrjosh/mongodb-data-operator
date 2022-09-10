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
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"k8s.io/apimachinery/pkg/runtime"
	// nolint
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	"github.com/mrjosh/mongodb-data-operator/pkg/mongodb"
	"github.com/operator-framework/operator-lib/handler"
	"github.com/pingcap/errors"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	mongoDBDataFinalizerName = "mongo.snappcloud.io/mongodb-data-finalizer"
	histogramVec             = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "mongodb_data_latency_histogram",
		Help: "Histogram of response time for Reconcile in seconds",
	}, []string{"name", "state"})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(histogramVec)
}

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

	histogramTimeStart := time.Now()

	log := r.Log.WithValues("mongodb-data", req.NamespacedName)
	log.Info("Reconciling MongoDBData")

	mongoData := &mongov1.MongoDBData{}
	if err := r.Client.Get(ctx, req.NamespacedName, mongoData); err != nil {
		if errors.IsNotFound(err) {
			// don't requeue on deletions, which yield a non-found object
			log.Info("ignoring", "reason", "not found", "err", err)
		}
		return requeue(client.IgnoreNotFound(err))
	}

	defer func() {
		histogramVec.WithLabelValues(
			mongoData.ObjectMeta.Name,
			mongoData.Status.State,
		).Observe(time.Since(histogramTimeStart).Seconds())
	}()

	// Check if the MongoDBConfig exists
	mongoCfg := &mongov1.MongoDBConfig{}
	if err := r.Client.Get(ctx, k8sTypes.NamespacedName{Name: mongoData.Spec.DB}, mongoCfg); err != nil {

		if errors.IsNotFound(err) {

			message := fmt.Sprintf("MongoDBConfig with name %s doesn't exists", mongoData.Spec.DB)
			if err := r.setEventStatusPending(ctx, mongoData, message); err != nil {
				log.Error(err, "unable to update target's status object")
				return requeue(err)
			}

			log.Info("ignoring", "reason", message)
			return requeueWithDelay(20 * time.Second)
		}

		log.Error(err, fmt.Sprintf("failed to get the mongo-config %s", mongoData.Spec.DB))
		return requeue(err)
	}

	if mongoData.Status.State == "" {

		// The object is being deleted
		mongoData.Status.State = string(mongov1.MongoDBDataConditionPending)
		if err := r.Client.Status().Update(ctx, mongoData); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// create a new mongodb client
	mongoClient, err := mongodb.NewClient(mongoCfg.Spec.MongoURL)
	if err != nil {

		if err := r.setEventStatusPending(ctx, mongoData, err.Error()); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return requeueWithDelay(30 * time.Second)
	}

	var (
		// using resource namespace as database name
		db         = mongoClient.Database(req.Namespace)
		collection = db.Collection(mongoCfg.Spec.Collection)
	)

	// examine DeletionTimestamp to determine if object is under deletion
	if mongoData.ObjectMeta.DeletionTimestamp.IsZero() {

		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(mongoData, mongoDBDataFinalizerName) {

			if controllerutil.AddFinalizer(mongoData, mongoDBDataFinalizerName) {

				if err := r.Client.Update(ctx, mongoData); err != nil {
					log.Error(err, "unable to update target")
					return requeue(err)
				}
			}

		}

	} else {

		if controllerutil.ContainsFinalizer(mongoData, mongoDBDataFinalizerName) {

			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteDocument(ctx, collection, mongoData); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				log.Error(err, "unable to remove object from mongodb")
				return requeue(err)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(mongoData, mongoDBDataFinalizerName)
			if err := r.Client.Update(ctx, mongoData); err != nil {
				log.Error(err, "unable to update target")
				return requeue(err)
			}

		}

		// Stop reconciliation as the item is being deleted
		return doNotRequeue()
	}

	// marshal the current MongoDBData object into bson.M for further mongodb operations
	data, err := bson.Marshal(mongoData.Spec.Data)
	if err != nil {

		if err := r.setEventStatusPending(ctx, mongoData, err.Error()); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return requeueWithDelay(30 * time.Second)
	}

	// check if mongodbData state is Inserted and there is a record on database
	// then update the record
	if mongoData.Status.State == string(mongov1.MongoDBDataConditionInserted) {

		// if document exists on database, MongoDBData.Status.ObjectID should not be empty
		// then we should update the document
		if mongoData.Status.ObjectID != "" {
			return r.findAndUpdateDocumentIfNeeded(ctx, log, collection, mongoData, mongoCfg, data)
		}
	}

	if mongoData.Status.State == string(mongov1.MongoDBDataConditionPending) {

		// check if mongodbData state is not Inserted, insert the document to mongodb collection
		return r.insertDocument(ctx, log, collection, mongoData, mongoCfg, data)
	}

	return doNotRequeue()
}

// updateDocument will update the current MongoDBData document from database
func (r *MongoDBDataReconciler) findAndUpdateDocumentIfNeeded(
	ctx context.Context,
	log logr.Logger,
	collection *mongo.Collection,
	mongoData *mongov1.MongoDBData,
	mongoCfg *mongov1.MongoDBConfig,
	document []byte,
) (ctrl.Result, error) {

	updateID, err := primitive.ObjectIDFromHex(mongoData.Status.ObjectID)
	if err != nil {
		log.Error(err, "unable to decode hex ObjectID to primitive.ObjectID")
		return requeue(err)
	}

	curser := collection.FindOne(ctx, bson.M{"_id": updateID})

	// decode find spec.Data into bson.M for comparison
	var find bson.M
	if err := curser.Decode(&find); err != nil {
		log.Error(err, "could not unmarshal mongodb find document into bson.M")
		return requeue(err)
	}

	// decode resource spec.Data into bson.M for comparison
	var specData bson.M
	if err := bson.Unmarshal(document, &specData); err != nil {
		log.Error(err, "could not unmarshal spec.data bson bytes into bson.M")
		return requeue(err)
	}

	// if current object is not equel with database document
	// we should update the document on db
	if !reflect.DeepEqual(find, specData) {

		result, err := collection.UpdateByID(ctx, updateID, bson.M{"$set": specData})
		if err != nil {

			if err := r.setEventStatusFailed(ctx, mongoData, err.Error()); err != nil {
				log.Error(err, "unable to update target's status object")
				return requeue(err)
			}

			return requeueWithDelay(30 * time.Second)
		}

		if result.ModifiedCount == 1 {

			msg := "Document updated successfully"
			if err := r.setEventStatusInserted(ctx, mongoData, msg); err != nil {

				log.Error(err, "unable to update target's status object")
				return requeue(err)
			}
		}

	}

	return doNotRequeue()
}

// insertDocument will insert current MongoDBData document into database
func (r *MongoDBDataReconciler) insertDocument(
	ctx context.Context,
	log logr.Logger,
	collection *mongo.Collection,
	mongoData *mongov1.MongoDBData,
	mongoCfg *mongov1.MongoDBConfig,
	document []byte,
) (ctrl.Result, error) {

	fmt.Println(mongoData.Status.ObjectID)

	// check if we have the document with ObjectID, then ignore the insert
	if mongoData.Status.ObjectID != "" {

		findOID, err := primitive.ObjectIDFromHex(mongoData.Status.ObjectID)
		if err != nil {
			log.Error(err, "unable to decode object_id")
			return requeue(client.IgnoreNotFound(err))
		}

		count, err := collection.CountDocuments(ctx, bson.M{"_id": findOID})
		if err != nil {
			log.Error(err, "unable to count document with ObjectID ", findOID)
			return requeue(client.IgnoreNotFound(err))
		}

		fmt.Println(count)

		if count > 0 {
			return doNotRequeue()
		}
	}

	result, err := collection.InsertOne(ctx, document)
	if err != nil {

		if err := r.setEventStatusFailed(ctx, mongoData, err.Error()); err != nil {
			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}

		return requeueWithDelay(30 * time.Second)
	}

	if result.InsertedID != nil {

		mongoData.Status.ObjectID = result.InsertedID.(primitive.ObjectID).Hex()
		msg := fmt.Sprintf("MongoDBData successfully Inserted into %s collection", mongoCfg.Spec.Collection)
		if err := r.setEventStatusInserted(ctx, mongoData, msg); err != nil {

			log.Error(err, "unable to update target's status object")
			return requeue(err)
		}
	}

	return doNotRequeue()
}

// deleteDocument will delete the current MongoDBData document from database
func (r *MongoDBDataReconciler) deleteDocument(ctx context.Context, coll *mongo.Collection, adapter *mongov1.MongoDBData) (err error) {
	objectID, err := primitive.ObjectIDFromHex(adapter.Status.ObjectID)
	if err != nil {
		return err
	}
	_, err = coll.DeleteOne(ctx, bson.M{"_id": objectID})
	return client.IgnoreNotFound(err)
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDataReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mongov1.MongoDBData{}).
		// registers an InstrumentedEnqueueRequest prometheus metric for mongov1.MongoDBData
		Watches(&source.Kind{Type: &mongov1.MongoDBData{}}, &handler.InstrumentedEnqueueRequestForObject{}).
		Complete(r)
}
