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
	"context"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	mongoURLRegexPattern = `(?m)(?m)mongodb:\/\/(?:(?:[^:]+):(?:[^@]+)?@)?(?:(?:(?:[^\/]+)|(?:\/.+.sock?),?)+)(?:\/([^\/\."*<>:\|\?]*))?(?:\?(?:(.+=.+)&?)+)*`
	mongodbconfiglog     = logf.Log.WithName("mongodbconfig-resource")
	kubeClient           client.Client
)

func (r *MongoDBConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kubeClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-mongo-snappcloud-io-v1-mongodbconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=mongo.snappcloud.io,resources=mongodbconfigs,verbs=create;update;delete,versions=v1,name=vmongodbconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MongoDBConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateCreate() error {
	mongodbconfiglog.Info("validate create", "name", r.ObjectMeta.Name)

	if err := r.validateSpecs(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateUpdate(old runtime.Object) error {
	mongodbconfiglog.Info("validate update", "name", r.ObjectMeta.Name)

	if err := r.validateSpecs(); err != nil {
		return newError(r.ObjectMeta.Name, err)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MongoDBConfig) ValidateDelete() error {
	mongodbconfiglog.Info("validate delete", "name", r.ObjectMeta.Name)

	key := field.NewPath("spec").Child("db")

	mongoDatasList := &MongoDBDataList{}
	if err := kubeClient.List(context.Background(), mongoDatasList); err != nil {
		return field.InternalError(key, err)
	}

	mongoCfgResource := fmt.Sprintf("MongoDBConfig/%s", r.ObjectMeta.Name)

	for _, md := range mongoDatasList.Items {
		if md.Spec.DB == r.ObjectMeta.Name {

			resourceNameWithNamespace := fmt.Sprintf(
				"%s/%s",
				md.ObjectMeta.Namespace,
				md.ObjectMeta.Name,
			)

			return field.Forbidden(key, fmt.Sprintf(
				"%s is using %s resource, consider removing %s first",
				resourceNameWithNamespace,
				mongoCfgResource,
				resourceNameWithNamespace,
			))
		}
	}

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
