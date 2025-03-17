//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v2beta1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Authority) DeepCopyInto(out *Authority) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Authority.
func (in *Authority) DeepCopy() *Authority {
	if in == nil {
		return nil
	}
	out := new(Authority)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Autoscaling) DeepCopyInto(out *Autoscaling) {
	*out = *in
	if in.AverageCPUUtilization != nil {
		in, out := &in.AverageCPUUtilization, &out.AverageCPUUtilization
		*out = new(int)
		**out = **in
	}
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int)
		**out = **in
	}
	if in.MaxReplicas != nil {
		in, out := &in.MaxReplicas, &out.MaxReplicas
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Autoscaling.
func (in *Autoscaling) DeepCopy() *Autoscaling {
	if in == nil {
		return nil
	}
	out := new(Autoscaling)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	if in.AnsibleResult != nil {
		in, out := &in.AnsibleResult, &out.AnsibleResult
		*out = new(ResultAnsible)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigMapRef) DeepCopyInto(out *ConfigMapRef) {
	*out = *in
	if in.Keys != nil {
		in, out := &in.Keys, &out.Keys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigMapRef.
func (in *ConfigMapRef) DeepCopy() *ConfigMapRef {
	if in == nil {
		return nil
	}
	out := new(ConfigMapRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Data) DeepCopyInto(out *Data) {
	*out = *in
	if in.GPKG != nil {
		in, out := &in.GPKG, &out.GPKG
		*out = new(GPKG)
		(*in).DeepCopyInto(*out)
	}
	if in.Postgis != nil {
		in, out := &in.Postgis, &out.Postgis
		*out = new(Postgis)
		(*in).DeepCopyInto(*out)
	}
	if in.Tif != nil {
		in, out := &in.Tif, &out.Tif
		*out = new(Tif)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Data.
func (in *Data) DeepCopy() *Data {
	if in == nil {
		return nil
	}
	out := new(Data)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FeatureType) DeepCopyInto(out *FeatureType) {
	*out = *in
	if in.Keywords != nil {
		in, out := &in.Keywords, &out.Keywords
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Extent != nil {
		in, out := &in.Extent, &out.Extent
		*out = new(string)
		**out = **in
	}
	in.Data.DeepCopyInto(&out.Data)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FeatureType.
func (in *FeatureType) DeepCopy() *FeatureType {
	if in == nil {
		return nil
	}
	out := new(FeatureType)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GPKG) DeepCopyInto(out *GPKG) {
	*out = *in
	if in.Columns != nil {
		in, out := &in.Columns, &out.Columns
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Aliases != nil {
		in, out := &in.Aliases, &out.Aliases
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GPKG.
func (in *GPKG) DeepCopy() *GPKG {
	if in == nil {
		return nil
	}
	out := new(GPKG)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *General) DeepCopyInto(out *General) {
	*out = *in
	if in.Theme != nil {
		in, out := &in.Theme, &out.Theme
		*out = new(string)
		**out = **in
	}
	if in.ServiceVersion != nil {
		in, out := &in.ServiceVersion, &out.ServiceVersion
		*out = new(string)
		**out = **in
	}
	if in.DataVersion != nil {
		in, out := &in.DataVersion, &out.DataVersion
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new General.
func (in *General) DeepCopy() *General {
	if in == nil {
		return nil
	}
	out := new(General)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheck) DeepCopyInto(out *HealthCheck) {
	*out = *in
	if in.Querystring != nil {
		in, out := &in.Querystring, &out.Querystring
		*out = new(string)
		**out = **in
	}
	if in.Mimetype != nil {
		in, out := &in.Mimetype, &out.Mimetype
		*out = new(string)
		**out = **in
	}
	if in.Boundingbox != nil {
		in, out := &in.Boundingbox, &out.Boundingbox
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheck.
func (in *HealthCheck) DeepCopy() *HealthCheck {
	if in == nil {
		return nil
	}
	out := new(HealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kubernetes) DeepCopyInto(out *Kubernetes) {
	*out = *in
	if in.Autoscaling != nil {
		in, out := &in.Autoscaling, &out.Autoscaling
		*out = new(Autoscaling)
		(*in).DeepCopyInto(*out)
	}
	if in.HealthCheck != nil {
		in, out := &in.HealthCheck, &out.HealthCheck
		*out = new(HealthCheck)
		(*in).DeepCopyInto(*out)
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Lifecycle != nil {
		in, out := &in.Lifecycle, &out.Lifecycle
		*out = new(Lifecycle)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kubernetes.
func (in *Kubernetes) DeepCopy() *Kubernetes {
	if in == nil {
		return nil
	}
	out := new(Kubernetes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LegendFile) DeepCopyInto(out *LegendFile) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LegendFile.
func (in *LegendFile) DeepCopy() *LegendFile {
	if in == nil {
		return nil
	}
	out := new(LegendFile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Lifecycle) DeepCopyInto(out *Lifecycle) {
	*out = *in
	if in.TTLInDays != nil {
		in, out := &in.TTLInDays, &out.TTLInDays
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Lifecycle.
func (in *Lifecycle) DeepCopy() *Lifecycle {
	if in == nil {
		return nil
	}
	out := new(Lifecycle)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Mapfile) DeepCopyInto(out *Mapfile) {
	*out = *in
	in.ConfigMapKeyRef.DeepCopyInto(&out.ConfigMapKeyRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Mapfile.
func (in *Mapfile) DeepCopy() *Mapfile {
	if in == nil {
		return nil
	}
	out := new(Mapfile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Postgis) DeepCopyInto(out *Postgis) {
	*out = *in
	if in.Columns != nil {
		in, out := &in.Columns, &out.Columns
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Aliases != nil {
		in, out := &in.Aliases, &out.Aliases
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Postgis.
func (in *Postgis) DeepCopy() *Postgis {
	if in == nil {
		return nil
	}
	out := new(Postgis)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resources) DeepCopyInto(out *Resources) {
	*out = *in
	if in.APIVersion != nil {
		in, out := &in.APIVersion, &out.APIVersion
		*out = new(string)
		**out = **in
	}
	if in.Kind != nil {
		in, out := &in.Kind, &out.Kind
		*out = new(string)
		**out = **in
	}
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resources.
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResultAnsible) DeepCopyInto(out *ResultAnsible) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResultAnsible.
func (in *ResultAnsible) DeepCopy() *ResultAnsible {
	if in == nil {
		return nil
	}
	out := new(ResultAnsible)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Deployment != nil {
		in, out := &in.Deployment, &out.Deployment
		*out = new(string)
		**out = **in
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]Resources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Style) DeepCopyInto(out *Style) {
	*out = *in
	if in.Title != nil {
		in, out := &in.Title, &out.Title
		*out = new(string)
		**out = **in
	}
	if in.Abstract != nil {
		in, out := &in.Abstract, &out.Abstract
		*out = new(string)
		**out = **in
	}
	if in.Visualization != nil {
		in, out := &in.Visualization, &out.Visualization
		*out = new(string)
		**out = **in
	}
	if in.LegendFile != nil {
		in, out := &in.LegendFile, &out.LegendFile
		*out = new(LegendFile)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Style.
func (in *Style) DeepCopy() *Style {
	if in == nil {
		return nil
	}
	out := new(Style)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StylingAssets) DeepCopyInto(out *StylingAssets) {
	*out = *in
	if in.ConfigMapRefs != nil {
		in, out := &in.ConfigMapRefs, &out.ConfigMapRefs
		*out = make([]ConfigMapRef, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.BlobKeys != nil {
		in, out := &in.BlobKeys, &out.BlobKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StylingAssets.
func (in *StylingAssets) DeepCopy() *StylingAssets {
	if in == nil {
		return nil
	}
	out := new(StylingAssets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tif) DeepCopyInto(out *Tif) {
	*out = *in
	if in.GetFeatureInfoIncludesClass != nil {
		in, out := &in.GetFeatureInfoIncludesClass, &out.GetFeatureInfoIncludesClass
		*out = new(bool)
		**out = **in
	}
	if in.Offsite != nil {
		in, out := &in.Offsite, &out.Offsite
		*out = new(string)
		**out = **in
	}
	if in.Resample != nil {
		in, out := &in.Resample, &out.Resample
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tif.
func (in *Tif) DeepCopy() *Tif {
	if in == nil {
		return nil
	}
	out := new(Tif)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WFS) DeepCopyInto(out *WFS) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(Status)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WFS.
func (in *WFS) DeepCopy() *WFS {
	if in == nil {
		return nil
	}
	out := new(WFS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WFS) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WFSList) DeepCopyInto(out *WFSList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WFS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WFSList.
func (in *WFSList) DeepCopy() *WFSList {
	if in == nil {
		return nil
	}
	out := new(WFSList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WFSList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WFSService) DeepCopyInto(out *WFSService) {
	*out = *in
	if in.Keywords != nil {
		in, out := &in.Keywords, &out.Keywords
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Authority = in.Authority
	if in.Extent != nil {
		in, out := &in.Extent, &out.Extent
		*out = new(string)
		**out = **in
	}
	if in.Maxfeatures != nil {
		in, out := &in.Maxfeatures, &out.Maxfeatures
		*out = new(string)
		**out = **in
	}
	if in.FeatureTypes != nil {
		in, out := &in.FeatureTypes, &out.FeatureTypes
		*out = make([]FeatureType, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Mapfile != nil {
		in, out := &in.Mapfile, &out.Mapfile
		*out = new(Mapfile)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WFSService.
func (in *WFSService) DeepCopy() *WFSService {
	if in == nil {
		return nil
	}
	out := new(WFSService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WFSSpec) DeepCopyInto(out *WFSSpec) {
	*out = *in
	in.General.DeepCopyInto(&out.General)
	in.Service.DeepCopyInto(&out.Service)
	in.Kubernetes.DeepCopyInto(&out.Kubernetes)
	in.Options.DeepCopyInto(&out.Options)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WFSSpec.
func (in *WFSSpec) DeepCopy() *WFSSpec {
	if in == nil {
		return nil
	}
	out := new(WFSSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMS) DeepCopyInto(out *WMS) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(Status)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMS.
func (in *WMS) DeepCopy() *WMS {
	if in == nil {
		return nil
	}
	out := new(WMS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WMS) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMSLayer) DeepCopyInto(out *WMSLayer) {
	*out = *in
	if in.Group != nil {
		in, out := &in.Group, &out.Group
		*out = new(string)
		**out = **in
	}
	if in.Title != nil {
		in, out := &in.Title, &out.Title
		*out = new(string)
		**out = **in
	}
	if in.Abstract != nil {
		in, out := &in.Abstract, &out.Abstract
		*out = new(string)
		**out = **in
	}
	if in.Keywords != nil {
		in, out := &in.Keywords, &out.Keywords
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DatasetMetadataIdentifier != nil {
		in, out := &in.DatasetMetadataIdentifier, &out.DatasetMetadataIdentifier
		*out = new(string)
		**out = **in
	}
	if in.SourceMetadataIdentifier != nil {
		in, out := &in.SourceMetadataIdentifier, &out.SourceMetadataIdentifier
		*out = new(string)
		**out = **in
	}
	if in.Styles != nil {
		in, out := &in.Styles, &out.Styles
		*out = make([]Style, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Extent != nil {
		in, out := &in.Extent, &out.Extent
		*out = new(string)
		**out = **in
	}
	if in.MinScale != nil {
		in, out := &in.MinScale, &out.MinScale
		*out = new(float64)
		**out = **in
	}
	if in.MaxScale != nil {
		in, out := &in.MaxScale, &out.MaxScale
		*out = new(float64)
		**out = **in
	}
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = new(Data)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMSLayer.
func (in *WMSLayer) DeepCopy() *WMSLayer {
	if in == nil {
		return nil
	}
	out := new(WMSLayer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMSList) DeepCopyInto(out *WMSList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WMS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMSList.
func (in *WMSList) DeepCopy() *WMSList {
	if in == nil {
		return nil
	}
	out := new(WMSList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WMSList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMSService) DeepCopyInto(out *WMSService) {
	*out = *in
	if in.Keywords != nil {
		in, out := &in.Keywords, &out.Keywords
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Authority = in.Authority
	if in.Layers != nil {
		in, out := &in.Layers, &out.Layers
		*out = make([]WMSLayer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Extent != nil {
		in, out := &in.Extent, &out.Extent
		*out = new(string)
		**out = **in
	}
	if in.Maxsize != nil {
		in, out := &in.Maxsize, &out.Maxsize
		*out = new(float64)
		**out = **in
	}
	if in.Resolution != nil {
		in, out := &in.Resolution, &out.Resolution
		*out = new(int)
		**out = **in
	}
	if in.DefResolution != nil {
		in, out := &in.DefResolution, &out.DefResolution
		*out = new(int)
		**out = **in
	}
	if in.StylingAssets != nil {
		in, out := &in.StylingAssets, &out.StylingAssets
		*out = new(StylingAssets)
		(*in).DeepCopyInto(*out)
	}
	if in.Mapfile != nil {
		in, out := &in.Mapfile, &out.Mapfile
		*out = new(Mapfile)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMSService.
func (in *WMSService) DeepCopy() *WMSService {
	if in == nil {
		return nil
	}
	out := new(WMSService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMSSpec) DeepCopyInto(out *WMSSpec) {
	*out = *in
	in.General.DeepCopyInto(&out.General)
	in.Service.DeepCopyInto(&out.Service)
	in.Options.DeepCopyInto(&out.Options)
	in.Kubernetes.DeepCopyInto(&out.Kubernetes)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMSSpec.
func (in *WMSSpec) DeepCopy() *WMSSpec {
	if in == nil {
		return nil
	}
	out := new(WMSSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WMSWFSOptions) DeepCopyInto(out *WMSWFSOptions) {
	*out = *in
	if in.ValidateRequests != nil {
		in, out := &in.ValidateRequests, &out.ValidateRequests
		*out = new(bool)
		**out = **in
	}
	if in.RewriteGroupToDataLayers != nil {
		in, out := &in.RewriteGroupToDataLayers, &out.RewriteGroupToDataLayers
		*out = new(bool)
		**out = **in
	}
	if in.DisableWebserviceProxy != nil {
		in, out := &in.DisableWebserviceProxy, &out.DisableWebserviceProxy
		*out = new(bool)
		**out = **in
	}
	if in.PrefetchData != nil {
		in, out := &in.PrefetchData, &out.PrefetchData
		*out = new(bool)
		**out = **in
	}
	if in.ValidateChildStyleNameEqual != nil {
		in, out := &in.ValidateChildStyleNameEqual, &out.ValidateChildStyleNameEqual
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WMSWFSOptions.
func (in *WMSWFSOptions) DeepCopy() *WMSWFSOptions {
	if in == nil {
		return nil
	}
	out := new(WMSWFSOptions)
	in.DeepCopyInto(out)
	return out
}
