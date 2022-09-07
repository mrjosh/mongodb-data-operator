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
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	mongoURLRegexPattern = `(?m)(?m)mongodb:\/\/(?:(?:[^:]+):(?:[^@]+)?@)?(?:(?:(?:[^\/]+)|(?:\/.+.sock?),?)+)(?:\/([^\/\."*<>:\|\?]*))?(?:\?(?:(.+=.+)&?)+)*`
	mongodbconfiglog     = logf.Log.WithName("mongodbconfig-resource")
)

func (r *MongoDBConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-mongo-snappcloud-io-v1-mongodbconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=mongo.snappcloud.io,resources=mongodbconfigs,verbs=create;update,versions=v1,name=vmongodbconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MongoDBConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateCreate() error {
	mongodbconfiglog.Info("validate create", "name", r.Name)

	if err := r.validateSpecs(); err != nil {
		return newError(r.Name, err)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateUpdate(old runtime.Object) error {
	mongodbconfiglog.Info("validate update", "name", r.Name)

	if err := r.validateSpecs(); err != nil {
		return newError(r.Name, err)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateDelete() error {
	mongodbconfiglog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *MongoDBConfig) validateSpecs() *field.Error {

	{
		key := field.NewPath("spec").Child("collection")
		value := r.Spec.Collection

		if value == "" {
			return field.Invalid(key, value, "collection must be specified")
		}
	}

	{
		key := field.NewPath("spec").Child("mongourl")
		value := r.Spec.MongoURL
		if value == "" {
			return field.Invalid(key, value, "mongourl must be specified")
		}

		if !r.isValidMongoURL(value) {
			return field.Invalid(key, value, "mongourl must be a valid connection string url")
		}
	}

	return nil
}

func (r *MongoDBConfig) isValidMongoURL(mongoURL string) bool {
	re := regexp.MustCompile(mongoURLRegexPattern)
	return len(re.FindStringIndex(mongoURL)) > 0
}
