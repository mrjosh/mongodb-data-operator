package controllers

import (
	"context"

	mongov1 "github.com/mrjosh/mongodb-data-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *MongoDBConfigReconciler) setEventStatusCondition(
	ctx context.Context,
	adapter *mongov1.MongoDBConfig,
	reason mongov1.MongoDBConfigConditionType,
	status metav1.ConditionStatus,
	message string,
) error {

	eventType := corev1.EventTypeWarning
	if status == metav1.ConditionTrue {
		eventType = corev1.EventTypeNormal
	}

	r.Recorder.Event(
		adapter,
		eventType,
		string(status),
		message,
	)

	adapter.Status.Ready = string(status)
	apimeta.SetStatusCondition(&adapter.Status.Conditions, metav1.Condition{
		Type:               string(reason),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: adapter.Generation,
	})

	return r.Client.Status().Update(ctx, adapter)
}

func (r *MongoDBConfigReconciler) setEventStatusReady(ctx context.Context, adapter *mongov1.MongoDBConfig, msg string) error {
	return r.setEventStatusCondition(
		ctx,
		adapter,
		mongov1.Ready,
		metav1.ConditionTrue,
		msg,
	)
}

func (r *MongoDBConfigReconciler) setEventStatusError(ctx context.Context, adapter *mongov1.MongoDBConfig, msg string) error {
	return r.setEventStatusCondition(
		ctx,
		adapter,
		mongov1.ConnectError,
		metav1.ConditionFalse,
		msg,
	)
}

func (r *MongoDBDataReconciler) setEventStatusPending(ctx context.Context, adapter *mongov1.MongoDBData, msg string) error {
	return r.setEventStatusCondition(
		ctx,
		adapter,
		mongov1.MongoDBDataConditionPending,
		metav1.ConditionFalse,
		msg,
	)
}

func (r *MongoDBDataReconciler) setEventStatusInserted(ctx context.Context, adapter *mongov1.MongoDBData, msg string) error {
	return r.setEventStatusCondition(
		ctx,
		adapter,
		mongov1.MongoDBDataConditionInserted,
		metav1.ConditionTrue,
		msg,
	)
}

func (r *MongoDBDataReconciler) setEventStatusFailed(ctx context.Context, adapter *mongov1.MongoDBData, msg string) error {
	return r.setEventStatusCondition(
		ctx,
		adapter,
		mongov1.MongoDBDataConditionFailed,
		metav1.ConditionFalse,
		msg,
	)
}

func (r *MongoDBDataReconciler) setEventStatusCondition(
	ctx context.Context,
	adapter *mongov1.MongoDBData,
	reason mongov1.MongoDBDataConditionType,
	status metav1.ConditionStatus,
	message string,
) error {

	eventType := corev1.EventTypeWarning
	if status == metav1.ConditionTrue {
		eventType = corev1.EventTypeNormal
	}

	r.Recorder.Event(adapter, eventType, string(status), message)

	adapter.Status.State = string(reason)
	apimeta.SetStatusCondition(&adapter.Status.Conditions, metav1.Condition{
		Type:               string(reason),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: adapter.Generation,
	})

	return r.Client.Status().Update(ctx, adapter)
}
