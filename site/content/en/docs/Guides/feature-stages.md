---
title: "Feature Stages"
linkTitle: "Feature Stages"
date: 2019-09-26T01:20:41Z
weight: 5
description: >
  This page provides a description of the various stages that Agones features can be in, and the
  relative maturity and support level expected for each level.
---

## Supported Versions

Agones versions are expressed as x.y.z, where x is the major version, y is the minor version, and z is the patch version
, following [Semantic Versioning](http://semver.org/) terminology.

## Agones Features

A feature within Agones can be in `Alpha`, `Beta` or `Stable` stage.

## Feature Gates

`Alpha` and `Beta` features can be enabled or disabled through the `agones.featureGate` configuration option 
that can be found in the [Helm configuration]({{< ref "/docs/Installation/Install Agones/helm.md#configuration" >}}) documentation.

The current set of `alpha` and `beta` feature gates are:

| Feature | Default | Stage | Since |
|---------|---------|-------|-------|
| Multicluster Allocation<sup>*</sup> | Enabled | `Alpha` | 0.11.0 | 

<sup>*</sup>Multicluster Allocation was started before this process was in place, and therefore is enabled by default,
 and will not have a feature flag.

## Description of Stages

### Alpha

An `Alpha` feature means:

* Disabled by default.
* Might be buggy. Enabling the feature may expose bugs.
* Support for this feature may be dropped at any time without notice.
* The API may change in incompatible ways in a later software release without notice.
* Recommended for use only in short-lived testing clusters, due to increased risk of bugs and lack of long-term support.

{{< alert title="Note" color="info" >}}
Please do try `Alpha` features and give feedback on them. This is important to ensure less breaking changes
through the `Beta` period.
{{< /alert >}}

### Beta

A `Beta` feature means:

* Enabled by default, but able to be disabled through a feature gate.
* The feature is well tested. Enabling the feature is considered safe.
* Support for the overall feature will not be dropped, though details may change.
* The schema and/or semantics of objects may change in incompatible ways in a subsequent beta or stable releases. When
  this happens, we will provide instructions for migrating to the next version. This may require deleting, editing,
  and re-creating API objects. The editing process may require some thought. This may require downtime for
  applications that rely on the feature.
* Recommended for only non-business-critical uses because of potential for incompatible changes in subsequent releases.
  If you have multiple clusters that can be upgraded independently, you may be able to relax this restriction.

{{< alert title="Note" color="info" >}}
Note: Please do try `Beta` features and give feedback on them! After they exit beta, it may not be practical for us
to make more changes.
{{< /alert >}}

### Stable

A `Stable` feature means:

* The feature is enabled and the corresponding feature gate no longer exists.
* Stable versions of features will appear in released software for many subsequent versions.

## Feature Stage Indicators

There are a variety of features with Agones, how can we determine what stage each feature is in?

Below are indicators for each type of functionality that can be used to determine the feature stage for a given aspect
of Agones.

### Custom Resource Definitions (CRDs)

This refers to Kubernetes resource for Agones, such as `GameServer`, `Fleet` and `GameServerAllocation`.

#### New CRDs

For new resources, the stage of the resource will be indicated by the `apiVersion` of the resource.

For example: `apiVersion: "agones.dev/v1"` is a `stable` resource, `apiVersion: "agones.dev/v1beta1"` is a `beta`
 stage resource, and `apiVersion: "agones.dev/v1alpha1"` is an `alpha` stage resource.

#### New CRD attributes

`alpha` and `beta` attributes will have a corresponding `alpha` or `beta` parent element in their configuration to
delineate their feature stage.

If no such parent exists, this attribute is a `stable` feature.

For example, if we were to add a hypothetical `alpha` configuration option of `timeoutSeconds` to the `GameServer`
 resource, it would be configured like so:
 
```yaml
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  generateName: "simple-udp-"
spec:
  alpha: # this is the alpha feature block
    timeoutSeconds: 30
  ports:
  - name: default
    portPolicy: Dynamic
    containerPort: 7654
  template:
    spec:
      containers:
      - name: simple-udp
        image: gcr.io/agones-images/udp-server:0.15
``` 

As resource attributes progress through their stages, there may be breaking changes, as backward conversion between
 `alpha`, `beta` and `stable` positioning in the resource configuration may not be guaranteed.

### Agones Game Server SDK

Any `alpha` or `beta` Game Server SDK functionality will be a subpackage of the `sdk` package. For example
, functionality found in a `sdk.alphav1` package should be considered at the `alpha` feature stage.

Only experimental functionality will be found in any `alpha` and `beta` SDK packages, and as such may change as 
development occurs. 

As SDK features move to through feature stages towards `stable`, the previous version of the SDK API
will remain for at least one release to enable easy migration to the more stable feature stage (i.e. from `alpha
` -> `beta`, `beta` -> `stable`)

Any other SDK functionality not marked as `alpha` or `beta` is assumed to be `stable`.

### REST & gRPC APIs 

REST and gRPC API will have versioned paths where appropriate to indicate their feature stage.

For example, a REST API with a prefix of `v1alpha1` is an `alpha` stage feature: 
`http://api.example.com/v1alpha1/exampleaction`.

Similar to the SDK, any `alpha` or `beta` gRPC functionality will be a subpackage of the main API package.
For example, functionality found in a `api.alphav1` package should be considered at the `alpha` feature stage. 
