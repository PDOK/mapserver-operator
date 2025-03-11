/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v2beta1

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// ConvertTo converts this WMS (v2beta1) to the Hub version (v3).
func (src *WMS) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.WMS)
	log.Printf("ConvertTo: Converting WMS from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v2beta1 to v3
	return nil
}

// ConvertFrom converts the Hub version (v3) to this WMS (v2beta1).
//
//nolint:revive
func (dst *WMS) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.WMS)
	log.Printf("ConvertFrom: Converting WMS from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v3 to v2beta1
	return nil
}
