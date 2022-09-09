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
	"regexp"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	emailRegexPattern = `^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`
	mongodbdatalog    = logf.Log.WithName("mongodbdata-resource")
)

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
	mongodbdatalog.Info("validate create", "name", r.ObjectMeta.Name)

	if err := r.validateDatabase(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	if err := r.validateSpecs(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBData) ValidateUpdate(old runtime.Object) error {
	mongodbdatalog.Info("validate update", "name", r.ObjectMeta.Name)

	key := field.NewPath("spec").Child("db")
	value := r.Spec.DB

	if err := r.validateDatabase(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	oldmdbd, ok := old.(*MongoDBData)
	if !ok {
		return errors.New("runtime.Object should be a type of mongov1.MongoDBData")
	}

	if value != oldmdbd.Spec.DB {
		return field.Forbidden(key, "cannot have a change on db field")
	}

	if err := r.validateSpecs(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBData) ValidateDelete() error {
	mongodbdatalog.Info("validate delete", "name", r.ObjectMeta.Name)
	return nil
}

func (r *MongoDBData) validateSpecs() *field.Error {

	// Validate spec.data.lastname
	{
		key := field.NewPath("spec").Child("data").Child("lastname")
		value := r.Spec.Data.Lastname

		if value == "" {
			return field.Invalid(key, value, "lastname cannot be empty")
		}
	}

	// Validate spec.data.email
	{

		key := field.NewPath("spec").Child("data").Child("email")
		value := r.Spec.Data.Email

		if value != "" {
			if !r.isValidEmail(value) {
				return field.Invalid(key, value, "email is not valid")
			}
		}
	}

	return nil
}

func (r *MongoDBData) validateDatabase() *field.Error {
	key := field.NewPath("spec").Child("db")
	value := r.Spec.DB

	if value == "" {
		return field.Invalid(key, value, "db cannot be empty")
	}

	return nil
}

func newError(name string, err *field.Error) error {
	return apierrors.NewInvalid(GroupKind, name, field.ErrorList{err})
}

func (r *MongoDBData) isValidEmail(email string) bool {
	re := regexp.MustCompile(emailRegexPattern)
	return len(re.FindStringIndex(email)) > 0
}
