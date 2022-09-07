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

package v1

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var mongodbdatalog = logf.Log.WithName("mongodbdata-resource")

func (r *MongoDBData) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-mongo-snappcloud-io-v1-mongodbdata,mutating=false,failurePolicy=fail,sideEffects=None,groups=mongo.snappcloud.io,resources=mongodbdata,verbs=create;update,versions=v1,name=vmongodbdata.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MongoDBData{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBData) ValidateCreate() error {
	mongodbdatalog.Info("validate create", "name", r.Name)

	if err := r.validateDatabase(); err != nil {
		return newError(r.Name, err)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBData) ValidateUpdate(old runtime.Object) error {
	mongodbdatalog.Info("validate update", "name", r.Name)

	key := field.NewPath("spec").Child("db")
	value := r.Spec.DB

	if err := r.validateDatabase(); err != nil {
		return newError(r.Name, err)
	}

	oldmdbd, ok := old.(*MongoDBData)
	if !ok {
		return errors.New("runtime.Object should be a type of mongov1.MongoDBData")
	}

	if value != oldmdbd.Spec.DB {
		return field.Invalid(key, value, "Can not have a change of db field")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBData) ValidateDelete() error {
	mongodbdatalog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *MongoDBData) validateDatabase() *field.Error {
	key := field.NewPath("spec").Child("db")
	value := r.Spec.DB

	if value == "" {
		return field.Invalid(key, value, "db must be configured")
	}

	return nil
}

func newError(name string, err *field.Error) error {
	return apierrors.NewInvalid(GroupKind, name, field.ErrorList{err})
}
