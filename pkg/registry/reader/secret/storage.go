/*
Copyright AppsCode Inc. and Contributors.

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

package secret

import (
	"context"
	"errors"

	readerv1alpha1 "kubevault.dev/secrets-store-reader/apis/reader/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	mu "kmodules.xyz/client-go/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ssapi "sigs.k8s.io/secrets-store-csi-driver/apis/v1"
)

type Storage struct {
	kc        client.Client
	a         authorizer.Authorizer
	gr        schema.GroupResource
	convertor rest.TableConvertor
}

var (
	_ rest.GroupVersionKindProvider = &Storage{}
	_ rest.Scoper                   = &Storage{}
	_ rest.Lister                   = &Storage{}
	_ rest.Getter                   = &Storage{}
)

func NewStorage(kc client.Client, a authorizer.Authorizer) *Storage {
	s := &Storage{
		kc: kc,
		a:  a,
		gr: schema.GroupResource{
			Group:    "",
			Resource: "secrets",
		},
		convertor: rest.NewDefaultTableConvertor(schema.GroupResource{
			Group:    readerv1alpha1.GroupName,
			Resource: readerv1alpha1.ResourceSecrets,
		}),
	}
	return s
}

func (r *Storage) GroupVersionKind(_ schema.GroupVersion) schema.GroupVersionKind {
	return readerv1alpha1.GroupVersion.WithKind(readerv1alpha1.ResourceKindSecret)
}

func (r *Storage) NamespaceScoped() bool {
	return true
}

func (r *Storage) New() runtime.Object {
	return &readerv1alpha1.Secret{}
}

func (r *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}
	user, ok := apirequest.UserFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing user info")
	}

	attrs := authorizer.AttributesRecord{
		User:      user,
		Verb:      "get",
		Namespace: ns,
		APIGroup:  r.gr.Group,
		Resource:  r.gr.Resource,
		Name:      name,
	}
	decision, why, err := r.a.Authorize(ctx, attrs)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if decision != authorizer.DecisionAllow {
		return nil, apierrors.NewForbidden(r.gr, name, errors.New(why))
	}

	var spc ssapi.SecretProviderClass
	err = r.kc.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &spc)
	if err != nil {
		return nil, err
	}

	return r.toSecret(&spc), nil
}

func (r *Storage) toSecret(spc *ssapi.SecretProviderClass) *readerv1alpha1.Secret {
	result := readerv1alpha1.Secret{
		// TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: *spc.ObjectMeta.DeepCopy(),
	}
	result.UID = "sec-" + spc.GetUID()
	result.ManagedFields = nil
	result.OwnerReferences = nil
	result.Finalizers = nil
	delete(result.ObjectMeta.Annotations, mu.LastAppliedConfigAnnotation)

	return &result
}

func (r *Storage) NewList() runtime.Object {
	return &readerv1alpha1.SecretList{}
}

func (r *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}

	user, ok := apirequest.UserFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing user info")
	}

	attrs := authorizer.AttributesRecord{
		User:      user,
		Verb:      "get",
		Namespace: ns,
		APIGroup:  r.gr.Group,
		Resource:  r.gr.Resource,
		Name:      "",
	}

	opts := client.ListOptions{Namespace: ns}
	if options != nil {
		if options.LabelSelector != nil && !options.LabelSelector.Empty() {
			opts.LabelSelector = options.LabelSelector
		}
		if options.FieldSelector != nil && !options.FieldSelector.Empty() {
			opts.FieldSelector = options.FieldSelector
		}
		opts.Limit = options.Limit
		opts.Continue = options.Continue
	}

	var spcList ssapi.SecretProviderClassList
	err := r.kc.List(context.TODO(), &spcList, &opts)
	if err != nil {
		return nil, err
	}

	secrets := make([]readerv1alpha1.Secret, 0, len(spcList.Items))
	for _, spc := range spcList.Items {
		attrs.Name = spc.Name
		decision, _, err := r.a.Authorize(context.TODO(), attrs)
		if err != nil {
			return nil, apierrors.NewInternalError(err)
		}
		if decision != authorizer.DecisionAllow {
			continue
		}

		secret := r.toSecret(&spc)
		secrets = append(secrets, *secret)
	}

	result := readerv1alpha1.SecretList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: spcList.ListMeta,
		Items:    secrets,
	}

	return &result, err
}

func (r *Storage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.convertor.ConvertToTable(ctx, object, tableOptions)
}
