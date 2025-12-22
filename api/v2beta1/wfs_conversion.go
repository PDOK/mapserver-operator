/*
Copyright 2025.

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

package v2beta1

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// ConvertTo converts this WFS (v2beta1) to the Hub version (v3).
func (src *WFS) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.WFS)
	log.Printf("ConvertTo: Converting WFS from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v2beta1 to v3
	// Example: Copying Spec fields
	// dst.Spec.Size = src.Spec.Replicas

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}

// ConvertFrom converts the Hub version (v3) to this WFS (v2beta1).
func (dst *WFS) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.WFS)
	log.Printf("ConvertFrom: Converting WFS from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v3 to v2beta1
	// Example: Copying Spec fields
	// dst.Spec.Replicas = src.Spec.Size

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}
