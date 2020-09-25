/*
 * SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package resources

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const owner_annotation = "resources.gardener.cloud/owners"

func (this *AbstractObject) GetOwnerReference() *metav1.OwnerReference {
	return metav1.NewControllerRef(this.ObjectData, this.GroupVersionKind())
}

func GetAnnotatedOwners(data ObjectData) []string {
	s := data.GetAnnotations()[owner_annotation]
	r := []string{}
	if s != "" {
		for _, o := range strings.Split(data.GetAnnotations()[owner_annotation], ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				r = append(r, o)
			}
		}
	}
	return r
}

func (this *AbstractObject) AddOwner(obj Object) bool {
	return AddOwner(this, this.self.GetCluster().GetId(), obj)
}

func AddOwner(data ObjectData, clusterid string, obj Object) bool {
	owner := obj.ClusterKey()
	if owner.Namespace() == data.GetNamespace() && owner.Cluster() == clusterid {
		// add kubernetes owners reference
		return SetOwnerReference(data, obj.GetOwnerReference())
	} else {
		// maintain foreign references via annotations
		ref := owner.AsRefFor(clusterid)
		refs := GetAnnotatedOwners(data)
		for _, r := range refs {
			if ref == strings.TrimSpace(r) {
				return false
			}
		}
		refs = append(refs, ref)
		return SetAnnotation(data, owner_annotation, strings.Join(refs, ","))
	}
}

func (this *AbstractObject) RemoveOwner(obj Object) bool {
	owner := obj.ClusterKey()
	if owner.Namespace() == this.GetNamespace() && owner.Cluster() == this.self.GetCluster().GetId() {
		// remove kubernetes owners reference
		ref := obj.GetOwnerReference()
		refs := this.GetOwnerReferences()
		new := []metav1.OwnerReference{}
		for _, r := range refs {
			if r.UID != ref.UID {
				new = append(new, r)
			}
		}
		if len(new) == len(refs) {
			return false
		}
		this.SetOwnerReferences(refs)
		return true
	} else {
		// maintain foreign references via annotations
		ref := owner.AsRefFor(this.self.GetCluster().GetId())
		refs := GetAnnotatedOwners(this)
		new := []string{}
		for _, r := range refs {
			if ref != strings.TrimSpace(r) {
				new = append(new, r)
			}
		}
		if len(new) == len(refs) {
			return false
		}
		this.GetAnnotations()[owner_annotation] = strings.Join(new, ",")
		return true
	}
}

func (this *AbstractObject) GetOwners(kinds ...schema.GroupKind) ClusterObjectKeySet {
	result := ClusterObjectKeySet{}
	cluster := this.self.GetCluster().GetId()

	for _, r := range this.GetOwnerReferences() {
		gv, _ := schema.ParseGroupVersion(r.APIVersion)
		result.Add(NewClusterKey(cluster, NewGroupKind(gv.Group, r.Kind), this.GetNamespace(), r.Name))
	}
	for _, r := range GetAnnotatedOwners(this) {
		ref, err := ParseClusterObjectKey(this.self.GetCluster().GetId(), r)
		if err == nil {
			result.Add(ref)
		}
	}
	return FilterKeysByGroupKinds(result, kinds...)
}
