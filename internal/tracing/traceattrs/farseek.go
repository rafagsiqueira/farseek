// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package traceattrs

import (
	"go.opentelemetry.io/otel/attribute"
)

// This file contains some functions representing Farseek-specific semantic
// conventions, which we use alongside the general OpenTelemetry-specified
// semantic conventions.
//
// These functions tend to take strings that are expected to be the canonical
// string representation of some more specific type from elsewhere in Farseek,
// but we make the caller produce the string representation rather than doing it
// inline because this package needs to avoid importing any other packages
// from this codebase so that the rest of Farseek can use this package without
// creating import cycles.
//
// We only create functions in here for attribute names that we want to use
// consistently across many different callers. For one-off attribute names that
// are only used in a single kind of span, use the generic functions like
// [String], [StringSlice], etc, instead.

// FarseekProviderAddress returns an attribute definition for indicating
// which provider is relevant to a particular trace span.
//
// The given address should be the result of calling [addrs.Provider.String].
func FarseekProviderAddress(addr string) attribute.KeyValue {
	return attribute.String("farseek.provider.address", addr)
}

// FarseekProviderVersion returns an attribute definition for indicating
// which version of a provider is relevant to a particular trace span.
//
// The given address should be the result of calling
// [getproviders.Version.String]. This should typically be used alongside
// [FarseekProviderAddress] to indicate which provider the version number is
// for.
func FarseekProviderVersion(v string) attribute.KeyValue {
	return attribute.String("farseek.provider.version", v)
}

// FarseekTargetPlatform returns an attribute definition for indicating
// which target platform is relevant to a particular trace span.
//
// The given address should be the result of calling
// [getproviders.Platform.String].
func FarseekTargetPlatform(platform string) attribute.KeyValue {
	return attribute.String("farseek.target_platform", platform)
}

// FarseekModuleCallName returns an attribute definition for indicating
// the name of a module call that's relevant to a particular trace span.
//
// The given address should be something that would be valid in the
// [addrs.ModuleCall.Name] field.
func FarseekModuleCallName(name string) attribute.KeyValue {
	return attribute.String("farseek.module.name", name)
}

// FarseekModuleSource returns an attribute definition for indicating
// which module source address is relevant to a particular trace span.
//
// The given address should be the result of calling
// [addrs.ModuleSource.String], or any other syntax-compatible representation.
func FarseekModuleSource(addr string) attribute.KeyValue {
	return attribute.String("farseek.module.source", addr)
}

// FarseekModuleVersion returns an attribute definition for indicating
// which version of a module is relevant to a particular trace span.
//
// The given address should be either the result of calling
// [getproviders.Version.String], or the String method from the "Version" type
// from HashiCorp's "go-version" library.
func FarseekModuleVersion(v string) attribute.KeyValue {
	return attribute.String("farseek.module.version", v)
}

// FarseekOCIReferenceTag returns an attribute definition for indicating
// which OCI repository tag is relevant to a particular trace span.
func FarseekOCIReferenceTag(name string) attribute.KeyValue {
	return attribute.String("farseek.oci.reference.tag", name)
}

// FarseekOCIReferenceDigest returns an attribute definition for indicating
// which OCI digest reference is relevant to a particular trace span.
func FarseekOCIReferenceDigest(digest string) attribute.KeyValue {
	return attribute.String("farseek.oci.reference.digest", digest)
}

// FarseekOCIManifestMediaType returns an attribute definition for indicating
// which OCI manifest media type is relevant to a particular trace span.
func FarseekOCIManifestMediaType(typ string) attribute.KeyValue {
	return attribute.String("farseek.oci.manifest.media_type", typ)
}

// FarseekOCIManifestArtifactType returns an attribute definition for indicating
// which OCI manifest artifact type is relevant to a particular trace span.
func FarseekOCIManifestArtifactType(typ string) attribute.KeyValue {
	return attribute.String("farseek.oci.manifest.artifact_type", typ)
}

// FarseekOCIManifestSize returns an attribute definition for indicating
// the size in bytes of an OCI manifest that is relevant to a particular
// trace span.
func FarseekOCIManifestSize(size int64) attribute.KeyValue {
	return attribute.Int64("farseek.oci.manifest.size", size)
}

// FarseekOCIBlobDigest returns an attribute definition for indicating
// which OCI blob digest is relevant to a particular trace span.
func FarseekOCIBlobDigest(digest string) attribute.KeyValue {
	return attribute.String("farseek.oci.blob.digest", digest)
}

// FarseekOCIBlobMediaType returns an attribute definition for indicating
// which OCI blob media type is relevant to a particular trace span.
func FarseekOCIBlobMediaType(typ string) attribute.KeyValue {
	return attribute.String("farseek.oci.blob.media_type", typ)
}

// FarseekOCIBlobArtifactType returns an attribute definition for indicating
// which OCI blob artifact type is relevant to a particular trace span.
func FarseekOCIBlobArtifactType(typ string) attribute.KeyValue {
	return attribute.String("farseek.oci.blob.artifact_type", typ)
}

// FarseekOCIBlobSize returns an attribute definition for indicating
// the size in bytes of an OCI blob that is relevant to a particular
// trace span.
func FarseekOCIBlobSize(size int64) attribute.KeyValue {
	return attribute.Int64("farseek.oci.blob.size", size)
}

// FarseekOCIRegistryDomain returns an attribute definition for indicating
// which OCI registry domain name is relevant to a particular trace span.
func FarseekOCIRegistryDomain(domain string) attribute.KeyValue {
	return attribute.String("farseek.oci.registry.domain", domain)
}

// FarseekOCIRepositoryName returns an attribute definition for indicating
// which OCI repository is relevant to a particular trace span.
//
// The value of this should not include the registry domain name. Use a
// separate attribute built from [FarseekOCIRegistryDomain] for that.
func FarseekOCIRepositoryName(name string) attribute.KeyValue {
	return attribute.String("farseek.oci.repository.name", name)
}
