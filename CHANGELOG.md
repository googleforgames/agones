# Changelog

## [v0.5.0](https://github.com/GoogleCloudPlatform/agones/tree/v0.5.0) (2018-10-16)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.5.0-rc...v0.5.0)

**Fixed bugs:**

- Tutorial fails @ Step 5 due to RBAC issues if you have capital letters in your gcloud account name [\#282](https://github.com/GoogleCloudPlatform/agones/issues/282)

**Closed issues:**

- Release 0.5.0.rc [\#378](https://github.com/GoogleCloudPlatform/agones/issues/378)

**Merged pull requests:**

- Troubleshooting guide for issues with Agones. [\#384](https://github.com/GoogleCloudPlatform/agones/pull/384) ([markmandel](https://github.com/markmandel))
- Spec docs for FleetAutoscaler [\#381](https://github.com/GoogleCloudPlatform/agones/pull/381) ([markmandel](https://github.com/markmandel))
- Post 0.5.0-rc updates [\#380](https://github.com/GoogleCloudPlatform/agones/pull/380) ([markmandel](https://github.com/markmandel))

## [v0.5.0-rc](https://github.com/GoogleCloudPlatform/agones/tree/v0.5.0-rc) (2018-10-09)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.4.0...v0.5.0-rc)

**Implemented enhancements:**

- Improve support for developing in custom environments [\#348](https://github.com/GoogleCloudPlatform/agones/issues/348)
- Agones helm repo [\#285](https://github.com/GoogleCloudPlatform/agones/issues/285)
- Add Amazon EKS Agones Setup Instructions [\#372](https://github.com/GoogleCloudPlatform/agones/pull/372) ([GabeBigBoxVR](https://github.com/GabeBigBoxVR))
- Agones stable helm repository [\#361](https://github.com/GoogleCloudPlatform/agones/pull/361) ([Kuqd](https://github.com/Kuqd))
- Improve support for custom dev environments [\#349](https://github.com/GoogleCloudPlatform/agones/pull/349) ([victor-prodan](https://github.com/victor-prodan))
- FleetAutoScaler v0 [\#340](https://github.com/GoogleCloudPlatform/agones/pull/340) ([victor-prodan](https://github.com/victor-prodan))
- Forces restart when using tls generation. [\#338](https://github.com/GoogleCloudPlatform/agones/pull/338) ([Kuqd](https://github.com/Kuqd))

**Fixed bugs:**

- Fix loophole in game server initialization [\#354](https://github.com/GoogleCloudPlatform/agones/issues/354)
- Health messages logged with wrong severity [\#335](https://github.com/GoogleCloudPlatform/agones/issues/335)
- Helm upgrade and SSL certificates [\#309](https://github.com/GoogleCloudPlatform/agones/issues/309)
- Fix for race condition: Allocation of Deleting GameServers Possible [\#367](https://github.com/GoogleCloudPlatform/agones/pull/367) ([markmandel](https://github.com/markmandel))
- Map level to severity for stackdriver [\#363](https://github.com/GoogleCloudPlatform/agones/pull/363) ([Kuqd](https://github.com/Kuqd))
- Add ReadTimeout for e2e tests, otherwise this can hang forever. [\#359](https://github.com/GoogleCloudPlatform/agones/pull/359) ([markmandel](https://github.com/markmandel))
- Fixes race condition bug with Pod not being scheduled before Ready\(\) [\#357](https://github.com/GoogleCloudPlatform/agones/pull/357) ([markmandel](https://github.com/markmandel))
- Allocation is broken when using the generated go client [\#347](https://github.com/GoogleCloudPlatform/agones/pull/347) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- \[Vuln\] Update to Alpine 3.8.1 [\#355](https://github.com/GoogleCloudPlatform/agones/issues/355)
- Update Alpine version to 3.8.1 [\#364](https://github.com/GoogleCloudPlatform/agones/pull/364) ([fooock](https://github.com/fooock))

**Closed issues:**

- C++ SDK no destructor body [\#366](https://github.com/GoogleCloudPlatform/agones/issues/366)
- Release 0.4.0 [\#341](https://github.com/GoogleCloudPlatform/agones/issues/341)
- Update "Developing, Testing and Building Agones" tutorial with how to push updates to your test cluster [\#308](https://github.com/GoogleCloudPlatform/agones/issues/308)
- Use revive instead of gometalinter [\#237](https://github.com/GoogleCloudPlatform/agones/issues/237)
- Integrate a spell and/or grammar check into build system [\#187](https://github.com/GoogleCloudPlatform/agones/issues/187)
- Helm package CI [\#153](https://github.com/GoogleCloudPlatform/agones/issues/153)
- Use functional parameters in Controller creation [\#104](https://github.com/GoogleCloudPlatform/agones/issues/104)

**Merged pull requests:**

- Release 0.5.0.rc changes [\#379](https://github.com/GoogleCloudPlatform/agones/pull/379) ([markmandel](https://github.com/markmandel))
- Make WaitForFleetCondition take up to 5 minutes [\#377](https://github.com/GoogleCloudPlatform/agones/pull/377) ([markmandel](https://github.com/markmandel))
- Fix for flaky test TestControllerAddress [\#376](https://github.com/GoogleCloudPlatform/agones/pull/376) ([markmandel](https://github.com/markmandel))
- Fix typo [\#374](https://github.com/GoogleCloudPlatform/agones/pull/374) ([Maxpain177](https://github.com/Maxpain177))
- Update instructions for Minikube 0.29.0 [\#373](https://github.com/GoogleCloudPlatform/agones/pull/373) ([markmandel](https://github.com/markmandel))
- Update README.md [\#371](https://github.com/GoogleCloudPlatform/agones/pull/371) ([mohammedfakhar](https://github.com/mohammedfakhar))
- Remove c++ sdk destructor causing linker errors [\#369](https://github.com/GoogleCloudPlatform/agones/pull/369) ([nikibobi](https://github.com/nikibobi))
- Update README.md [\#362](https://github.com/GoogleCloudPlatform/agones/pull/362) ([mohammedfakhar](https://github.com/mohammedfakhar))
- Upgrade GKE version and increase test cluster size [\#360](https://github.com/GoogleCloudPlatform/agones/pull/360) ([markmandel](https://github.com/markmandel))
- Fix typo in sdk readme which said only two sdks [\#356](https://github.com/GoogleCloudPlatform/agones/pull/356) ([ReDucTor](https://github.com/ReDucTor))
- Add allocator service example and documentation [\#353](https://github.com/GoogleCloudPlatform/agones/pull/353) ([slartibaartfast](https://github.com/slartibaartfast))
- Adding goimports back into the build shell. [\#352](https://github.com/GoogleCloudPlatform/agones/pull/352) ([markmandel](https://github.com/markmandel))
- e2e tests for Fleet Scaling and Updates [\#351](https://github.com/GoogleCloudPlatform/agones/pull/351) ([markmandel](https://github.com/markmandel))
- Switch to golangci-lint [\#346](https://github.com/GoogleCloudPlatform/agones/pull/346) ([Kuqd](https://github.com/Kuqd))
- Prepare for next release - 0.5.0.rc [\#343](https://github.com/GoogleCloudPlatform/agones/pull/343) ([markmandel](https://github.com/markmandel))

## [v0.4.0](https://github.com/GoogleCloudPlatform/agones/tree/v0.4.0) (2018-09-04)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.4.0.rc...v0.4.0)

**Closed issues:**

- Release 0.4.0.rc [\#330](https://github.com/GoogleCloudPlatform/agones/issues/330)

**Merged pull requests:**

- Release 0.4.0 [\#342](https://github.com/GoogleCloudPlatform/agones/pull/342) ([markmandel](https://github.com/markmandel))
- Fix yaml file paths [\#339](https://github.com/GoogleCloudPlatform/agones/pull/339) ([oskoi](https://github.com/oskoi))
- Add Troubleshooting section to Build doc [\#337](https://github.com/GoogleCloudPlatform/agones/pull/337) ([victor-prodan](https://github.com/victor-prodan))
- Preparing for 0.4.0 release next week. [\#333](https://github.com/GoogleCloudPlatform/agones/pull/333) ([markmandel](https://github.com/markmandel))

## [v0.4.0.rc](https://github.com/GoogleCloudPlatform/agones/tree/v0.4.0.rc) (2018-08-28)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.3.0...v0.4.0.rc)

**Implemented enhancements:**

- When running the SDK sidecar in local mode, be able to specify the backing `GameServer` configuration [\#296](https://github.com/GoogleCloudPlatform/agones/issues/296)
- Move Status \> Address & Status \> Ports population to `Creating` state processing [\#293](https://github.com/GoogleCloudPlatform/agones/issues/293)
- Propagating game server process events to Agones system [\#279](https://github.com/GoogleCloudPlatform/agones/issues/279)
- Session data propagation to dedicated server [\#277](https://github.com/GoogleCloudPlatform/agones/issues/277)
- Ability to pass `GameServer` yaml/json to local sdk server [\#328](https://github.com/GoogleCloudPlatform/agones/pull/328) ([markmandel](https://github.com/markmandel))
- Move Status \> Address & Ports population to `Creating` state processing [\#326](https://github.com/GoogleCloudPlatform/agones/pull/326) ([markmandel](https://github.com/markmandel))
- Implement SDK SetLabel and SetAnnotation functionality [\#323](https://github.com/GoogleCloudPlatform/agones/pull/323) ([markmandel](https://github.com/markmandel))
- Implements SDK callback for GameServer updates [\#316](https://github.com/GoogleCloudPlatform/agones/pull/316) ([markmandel](https://github.com/markmandel))
- Features/e2e [\#315](https://github.com/GoogleCloudPlatform/agones/pull/315) ([Kuqd](https://github.com/Kuqd))
- Metadata propagation from fleet allocation to game server [\#312](https://github.com/GoogleCloudPlatform/agones/pull/312) ([victor-prodan](https://github.com/victor-prodan))

**Fixed bugs:**

- Fleet allocation request could not find fleet [\#324](https://github.com/GoogleCloudPlatform/agones/issues/324)
- Hotfix: Ensure multiple Pods don't get created for a GameServer [\#332](https://github.com/GoogleCloudPlatform/agones/pull/332) ([markmandel](https://github.com/markmandel))
- Fleet Allocation via REST was failing [\#325](https://github.com/GoogleCloudPlatform/agones/pull/325) ([markmandel](https://github.com/markmandel))
- Make sure the test-e2e ensures the build image. [\#322](https://github.com/GoogleCloudPlatform/agones/pull/322) ([markmandel](https://github.com/markmandel))
- Update getting started guides with kubectl custom columns [\#319](https://github.com/GoogleCloudPlatform/agones/pull/319) ([markmandel](https://github.com/markmandel))
- Fix bug: Disabled health checking not implemented [\#317](https://github.com/GoogleCloudPlatform/agones/pull/317) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.3.0 [\#304](https://github.com/GoogleCloudPlatform/agones/issues/304)
- Change container builder steps to run concurrently [\#186](https://github.com/GoogleCloudPlatform/agones/issues/186)
- Move Deployment in install script out of v1beta1 [\#173](https://github.com/GoogleCloudPlatform/agones/issues/173)
- YAML packaging [\#101](https://github.com/GoogleCloudPlatform/agones/issues/101)

**Merged pull requests:**

- Changelog, and documentation changes for 0.4.0.rc [\#331](https://github.com/GoogleCloudPlatform/agones/pull/331) ([markmandel](https://github.com/markmandel))
- Added github.com/spf13/viper to dep toml [\#327](https://github.com/GoogleCloudPlatform/agones/pull/327) ([markmandel](https://github.com/markmandel))
- Add Minikube instructions [\#321](https://github.com/GoogleCloudPlatform/agones/pull/321) ([slartibaartfast](https://github.com/slartibaartfast))
- Convert Go example into multi-stage Docker build [\#320](https://github.com/GoogleCloudPlatform/agones/pull/320) ([markmandel](https://github.com/markmandel))
- Removal of the legacy port configuration [\#318](https://github.com/GoogleCloudPlatform/agones/pull/318) ([markmandel](https://github.com/markmandel))
- Fix flakiness with TestSidecarHTTPHealthCheck [\#313](https://github.com/GoogleCloudPlatform/agones/pull/313) ([markmandel](https://github.com/markmandel))
- Move linting into it's own serial step [\#311](https://github.com/GoogleCloudPlatform/agones/pull/311) ([markmandel](https://github.com/markmandel))
- Update to move from release to the next version \(0.4.0.rc\) [\#306](https://github.com/GoogleCloudPlatform/agones/pull/306) ([markmandel](https://github.com/markmandel))

## [v0.3.0](https://github.com/GoogleCloudPlatform/agones/tree/v0.3.0) (2018-07-26)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.3.0.rc...v0.3.0)

**Fixed bugs:**

- Missing `watch` permission in rbac for agones-sdk [\#300](https://github.com/GoogleCloudPlatform/agones/pull/300) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.3.0.rc [\#290](https://github.com/GoogleCloudPlatform/agones/issues/290)

**Merged pull requests:**

- Changes for release  0.3.0 [\#305](https://github.com/GoogleCloudPlatform/agones/pull/305) ([markmandel](https://github.com/markmandel))
- Move back to 0.3.0 [\#292](https://github.com/GoogleCloudPlatform/agones/pull/292) ([markmandel](https://github.com/markmandel))

## [v0.3.0.rc](https://github.com/GoogleCloudPlatform/agones/tree/v0.3.0.rc) (2018-07-17)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.2.0...v0.3.0.rc)

**Breaking changes:**

- \[Breaking Change\] Multiple port support for `GameServer` [\#283](https://github.com/GoogleCloudPlatform/agones/pull/283) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Expose SDK Sidecar GRPC Server as HTTP+JSON [\#240](https://github.com/GoogleCloudPlatform/agones/issues/240)
- supporting multiple ports [\#151](https://github.com/GoogleCloudPlatform/agones/issues/151)
- Support Cluster Node addition/deletion [\#60](https://github.com/GoogleCloudPlatform/agones/issues/60)
- SDK `GameServer\(\)` function for retrieving backing GameServer configuration [\#288](https://github.com/GoogleCloudPlatform/agones/pull/288) ([markmandel](https://github.com/markmandel))
- Move cluster node addition/removal out of "experimental" [\#271](https://github.com/GoogleCloudPlatform/agones/pull/271) ([markmandel](https://github.com/markmandel))
- added information about Agones running on Azure Kubernetes Service [\#269](https://github.com/GoogleCloudPlatform/agones/pull/269) ([dgkanatsios](https://github.com/dgkanatsios))
- Expose SDK-Server at HTTP+JSON [\#265](https://github.com/GoogleCloudPlatform/agones/pull/265) ([markmandel](https://github.com/markmandel))
- Support Rust SDK by gRPC-rs [\#230](https://github.com/GoogleCloudPlatform/agones/pull/230) ([thara](https://github.com/thara))

**Fixed bugs:**

- Error running make install with GKE [\#258](https://github.com/GoogleCloudPlatform/agones/issues/258)
- Minikube does not start with 0.26.x [\#192](https://github.com/GoogleCloudPlatform/agones/issues/192)
- Forgot to update the k8s client-go codegen. [\#281](https://github.com/GoogleCloudPlatform/agones/pull/281) ([markmandel](https://github.com/markmandel))
- Fix bug with hung GameServer resource on Kubernetes 1.10 [\#278](https://github.com/GoogleCloudPlatform/agones/pull/278) ([markmandel](https://github.com/markmandel))
- Fix Xonotic example race condition [\#266](https://github.com/GoogleCloudPlatform/agones/pull/266) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Agones on Azure AKS [\#254](https://github.com/GoogleCloudPlatform/agones/issues/254)
- Release v0.2.0 [\#242](https://github.com/GoogleCloudPlatform/agones/issues/242)
- helm namespace [\#212](https://github.com/GoogleCloudPlatform/agones/issues/212)

**Merged pull requests:**

- Release 0.3.0.rc [\#291](https://github.com/GoogleCloudPlatform/agones/pull/291) ([markmandel](https://github.com/markmandel))
- Update README.md with information about Public IPs on AKS [\#289](https://github.com/GoogleCloudPlatform/agones/pull/289) ([dgkanatsios](https://github.com/dgkanatsios))
- fix yaml install link [\#286](https://github.com/GoogleCloudPlatform/agones/pull/286) ([nikibobi](https://github.com/nikibobi))
- install.yaml now installs by default in agones-system [\#284](https://github.com/GoogleCloudPlatform/agones/pull/284) ([Kuqd](https://github.com/Kuqd))
- Update GKE testing cluster to 1.10.5 [\#280](https://github.com/GoogleCloudPlatform/agones/pull/280) ([markmandel](https://github.com/markmandel))
- Update dependencies to support K8s 1.10.x [\#276](https://github.com/GoogleCloudPlatform/agones/pull/276) ([markmandel](https://github.com/markmandel))
- Remove line [\#274](https://github.com/GoogleCloudPlatform/agones/pull/274) ([markmandel](https://github.com/markmandel))
- Update minikube instructions to 0.28.0 [\#273](https://github.com/GoogleCloudPlatform/agones/pull/273) ([markmandel](https://github.com/markmandel))
- Added list of various libs used in code [\#272](https://github.com/GoogleCloudPlatform/agones/pull/272) ([mean-mango](https://github.com/mean-mango))
- More Docker and Kubernetes Getting Started Resources [\#270](https://github.com/GoogleCloudPlatform/agones/pull/270) ([markmandel](https://github.com/markmandel))
- Fixing Flaky test TestControllerSyncFleet [\#268](https://github.com/GoogleCloudPlatform/agones/pull/268) ([markmandel](https://github.com/markmandel))
- Update Helm App Version [\#267](https://github.com/GoogleCloudPlatform/agones/pull/267) ([markmandel](https://github.com/markmandel))
- Give linter 15 minutes. [\#264](https://github.com/GoogleCloudPlatform/agones/pull/264) ([markmandel](https://github.com/markmandel))
- Upgrade to Go 1.10.3 [\#263](https://github.com/GoogleCloudPlatform/agones/pull/263) ([markmandel](https://github.com/markmandel))
- Upgrade Helm for build tools [\#262](https://github.com/GoogleCloudPlatform/agones/pull/262) ([markmandel](https://github.com/markmandel))
- Fixed some links & typos [\#261](https://github.com/GoogleCloudPlatform/agones/pull/261) ([mean-mango](https://github.com/mean-mango))
- Flaky test fix: TestWorkQueueHealthCheck [\#260](https://github.com/GoogleCloudPlatform/agones/pull/260) ([markmandel](https://github.com/markmandel))
- Upgrade gRPC to 1.12.0 [\#259](https://github.com/GoogleCloudPlatform/agones/pull/259) ([markmandel](https://github.com/markmandel))
- Flakey test fix: TestControllerUpdateFleetStatus [\#257](https://github.com/GoogleCloudPlatform/agones/pull/257) ([markmandel](https://github.com/markmandel))
- Remove reference to internal console site. [\#256](https://github.com/GoogleCloudPlatform/agones/pull/256) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Add Licences to Rust SDK & Examples [\#253](https://github.com/GoogleCloudPlatform/agones/pull/253) ([markmandel](https://github.com/markmandel))
- Clearer Helm installation instructions [\#252](https://github.com/GoogleCloudPlatform/agones/pull/252) ([markmandel](https://github.com/markmandel))
- Rust SDK Doc additions [\#251](https://github.com/GoogleCloudPlatform/agones/pull/251) ([markmandel](https://github.com/markmandel))
- use the helm --namespace convention  [\#250](https://github.com/GoogleCloudPlatform/agones/pull/250) ([Kuqd](https://github.com/Kuqd))
- fix podspec template broken link to documentation [\#247](https://github.com/GoogleCloudPlatform/agones/pull/247) ([Kuqd](https://github.com/Kuqd))
- Make Cloud Builder Faster [\#245](https://github.com/GoogleCloudPlatform/agones/pull/245) ([markmandel](https://github.com/markmandel))
- Increment base version [\#244](https://github.com/GoogleCloudPlatform/agones/pull/244) ([markmandel](https://github.com/markmandel))
- Lock protoc-gen-go to 1.0 release [\#241](https://github.com/GoogleCloudPlatform/agones/pull/241) ([markmandel](https://github.com/markmandel))

## [v0.2.0](https://github.com/GoogleCloudPlatform/agones/tree/v0.2.0) (2018-06-06)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.2.0.rc...v0.2.0)

**Closed issues:**

- Release v0.2.0.rc [\#231](https://github.com/GoogleCloudPlatform/agones/issues/231)

**Merged pull requests:**

- Release 0.2.0 [\#243](https://github.com/GoogleCloudPlatform/agones/pull/243) ([markmandel](https://github.com/markmandel))
- Adding my streaming development to contributing [\#239](https://github.com/GoogleCloudPlatform/agones/pull/239) ([markmandel](https://github.com/markmandel))
- Updates to release process [\#235](https://github.com/GoogleCloudPlatform/agones/pull/235) ([markmandel](https://github.com/markmandel))
- Adding a README.md file for the simple-udp to help developer to get start [\#234](https://github.com/GoogleCloudPlatform/agones/pull/234) ([g-ericso](https://github.com/g-ericso))
- Revert install configuration back to 0.2.0 [\#233](https://github.com/GoogleCloudPlatform/agones/pull/233) ([markmandel](https://github.com/markmandel))

## [v0.2.0.rc](https://github.com/GoogleCloudPlatform/agones/tree/v0.2.0.rc) (2018-05-30)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/v0.1...v0.2.0.rc)

**Implemented enhancements:**

- Generate Certs for Mutation/Validatiion Webhooks [\#169](https://github.com/GoogleCloudPlatform/agones/issues/169)
- Add liveness check to `pkg/gameservers/controller`. [\#116](https://github.com/GoogleCloudPlatform/agones/issues/116)
- GameServer Fleets [\#70](https://github.com/GoogleCloudPlatform/agones/issues/70)
- Release steps of archiving installation resources and documentation [\#226](https://github.com/GoogleCloudPlatform/agones/pull/226) ([markmandel](https://github.com/markmandel))
- Lint timeout increase, and make configurable [\#221](https://github.com/GoogleCloudPlatform/agones/pull/221) ([markmandel](https://github.com/markmandel))
- add the ability to turn off RBAC in helm and customize gcp test-cluster [\#220](https://github.com/GoogleCloudPlatform/agones/pull/220) ([Kuqd](https://github.com/Kuqd))
- Target for generating a CHANGELOG from GitHub Milestones. [\#217](https://github.com/GoogleCloudPlatform/agones/pull/217) ([markmandel](https://github.com/markmandel))
- Generate Certs for Mutation/Validatiion Webhooks [\#214](https://github.com/GoogleCloudPlatform/agones/pull/214) ([Kuqd](https://github.com/Kuqd))
- Rolling updates for Fleets [\#213](https://github.com/GoogleCloudPlatform/agones/pull/213) ([markmandel](https://github.com/markmandel))
- helm namespaces [\#210](https://github.com/GoogleCloudPlatform/agones/pull/210) ([Kuqd](https://github.com/Kuqd))
- Fleet update strategy: Replace [\#199](https://github.com/GoogleCloudPlatform/agones/pull/199) ([markmandel](https://github.com/markmandel))
- Status \> AllocatedReplicas on Fleets & GameServers [\#196](https://github.com/GoogleCloudPlatform/agones/pull/196) ([markmandel](https://github.com/markmandel))
- Creating a FleetAllocation allocated a GameServer from a Fleet [\#193](https://github.com/GoogleCloudPlatform/agones/pull/193) ([markmandel](https://github.com/markmandel))
- Add nano as editor to the build image [\#179](https://github.com/GoogleCloudPlatform/agones/pull/179) ([markmandel](https://github.com/markmandel))
- Feature/gometalinter [\#176](https://github.com/GoogleCloudPlatform/agones/pull/176) ([EricFortin](https://github.com/EricFortin))
- Creating a Fleet creates a GameServerSet [\#174](https://github.com/GoogleCloudPlatform/agones/pull/174) ([markmandel](https://github.com/markmandel))
- Register liveness check in gameservers.Controller [\#160](https://github.com/GoogleCloudPlatform/agones/pull/160) ([enocom](https://github.com/enocom))
- GameServerSet Implementation [\#156](https://github.com/GoogleCloudPlatform/agones/pull/156) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- gometalinter fails [\#181](https://github.com/GoogleCloudPlatform/agones/issues/181)
- Line endings in Windows make the project can't be compiled [\#180](https://github.com/GoogleCloudPlatform/agones/issues/180)
- Missing links in documentation [\#165](https://github.com/GoogleCloudPlatform/agones/issues/165)
- Cannot run GameServer in non-default namespace [\#146](https://github.com/GoogleCloudPlatform/agones/issues/146)
- Don't allow allocation of Deleted GameServers [\#198](https://github.com/GoogleCloudPlatform/agones/pull/198) ([markmandel](https://github.com/markmandel))
- Fixes for GKE issues with install/quickstart [\#197](https://github.com/GoogleCloudPlatform/agones/pull/197) ([markmandel](https://github.com/markmandel))
- `minikube-test-cluster` needed the `ensure-build-image` dependency [\#194](https://github.com/GoogleCloudPlatform/agones/pull/194) ([markmandel](https://github.com/markmandel))
- Update initialClusterVersion to 1.9.6.gke.1 [\#190](https://github.com/GoogleCloudPlatform/agones/pull/190) ([markmandel](https://github.com/markmandel))
- Point the install.yaml to the release-0.1 branch [\#189](https://github.com/GoogleCloudPlatform/agones/pull/189) ([markmandel](https://github.com/markmandel))
- Fixed missing links in documentation. [\#166](https://github.com/GoogleCloudPlatform/agones/pull/166) ([fooock](https://github.com/fooock))

**Security fixes:**

- RBAC: controller doesn't need fleet create [\#202](https://github.com/GoogleCloudPlatform/agones/pull/202) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- helm RBAC on/off [\#211](https://github.com/GoogleCloudPlatform/agones/issues/211)
- Release cycle [\#203](https://github.com/GoogleCloudPlatform/agones/issues/203)
- Fix cyclomatic complexity in examples/simple-udp/server/main.go [\#178](https://github.com/GoogleCloudPlatform/agones/issues/178)
- Fix cyclomatic complexity in cmd/controller/main.go [\#177](https://github.com/GoogleCloudPlatform/agones/issues/177)
- Add .helmignore to Helm chart [\#168](https://github.com/GoogleCloudPlatform/agones/issues/168)
- Add gometalinter to build [\#163](https://github.com/GoogleCloudPlatform/agones/issues/163)
- Google Bot is double posting [\#155](https://github.com/GoogleCloudPlatform/agones/issues/155)
- Add .editorconfig to ensure common formatting [\#97](https://github.com/GoogleCloudPlatform/agones/issues/97)

**Merged pull requests:**

- Release v0.2.0.rc [\#232](https://github.com/GoogleCloudPlatform/agones/pull/232) ([markmandel](https://github.com/markmandel))
- do-release release registry and upstream push [\#228](https://github.com/GoogleCloudPlatform/agones/pull/228) ([markmandel](https://github.com/markmandel))
- Archive C++ src on build and release [\#227](https://github.com/GoogleCloudPlatform/agones/pull/227) ([markmandel](https://github.com/markmandel))
- Update installing\_agones.md [\#225](https://github.com/GoogleCloudPlatform/agones/pull/225) ([g-ericso](https://github.com/g-ericso))
- Some missing tasks in the release [\#224](https://github.com/GoogleCloudPlatform/agones/pull/224) ([markmandel](https://github.com/markmandel))
- Move to proper semver [\#223](https://github.com/GoogleCloudPlatform/agones/pull/223) ([markmandel](https://github.com/markmandel))
- Update tools - vetshadow is more aggressive [\#222](https://github.com/GoogleCloudPlatform/agones/pull/222) ([markmandel](https://github.com/markmandel))
- add helm ignore file [\#219](https://github.com/GoogleCloudPlatform/agones/pull/219) ([Kuqd](https://github.com/Kuqd))
- Intercept Xonotic stdout for SDK Integration [\#218](https://github.com/GoogleCloudPlatform/agones/pull/218) ([markmandel](https://github.com/markmandel))
- Some more Extending Kubernetes resources [\#216](https://github.com/GoogleCloudPlatform/agones/pull/216) ([markmandel](https://github.com/markmandel))
- Release process documentation [\#215](https://github.com/GoogleCloudPlatform/agones/pull/215) ([markmandel](https://github.com/markmandel))
- Fix cyclomatic complexity in cmd/controller/main.go [\#209](https://github.com/GoogleCloudPlatform/agones/pull/209) ([enocom](https://github.com/enocom))
- Testing functions for resource events [\#208](https://github.com/GoogleCloudPlatform/agones/pull/208) ([markmandel](https://github.com/markmandel))
- Shrink main func to resolve gocyclo warning [\#207](https://github.com/GoogleCloudPlatform/agones/pull/207) ([enocom](https://github.com/enocom))
- Clearer docs on developing and building from source [\#206](https://github.com/GoogleCloudPlatform/agones/pull/206) ([markmandel](https://github.com/markmandel))
- Add formatting guidelines to CONTRIBUTING.md [\#205](https://github.com/GoogleCloudPlatform/agones/pull/205) ([enocom](https://github.com/enocom))
- Fleet docs: Some missing pieces. [\#204](https://github.com/GoogleCloudPlatform/agones/pull/204) ([markmandel](https://github.com/markmandel))
- Release version, and twitter badges. [\#201](https://github.com/GoogleCloudPlatform/agones/pull/201) ([markmandel](https://github.com/markmandel))
- Typo in GameServer json [\#200](https://github.com/GoogleCloudPlatform/agones/pull/200) ([markmandel](https://github.com/markmandel))
- Install docs: minikube 0.25.2 and k8s 1.9.4 [\#195](https://github.com/GoogleCloudPlatform/agones/pull/195) ([markmandel](https://github.com/markmandel))
- Update temporary auth against Google Container Registry [\#191](https://github.com/GoogleCloudPlatform/agones/pull/191) ([markmandel](https://github.com/markmandel))
- Make the development release warning more visible. [\#188](https://github.com/GoogleCloudPlatform/agones/pull/188) ([markmandel](https://github.com/markmandel))
- Solve rare flakiness on TestWorkerQueueHealthy [\#185](https://github.com/GoogleCloudPlatform/agones/pull/185) ([markmandel](https://github.com/markmandel))
- Adding additional resources for CRDs and Controllers [\#184](https://github.com/GoogleCloudPlatform/agones/pull/184) ([markmandel](https://github.com/markmandel))
- Reworked some Dockerfiles to improve cache usage. [\#183](https://github.com/GoogleCloudPlatform/agones/pull/183) ([EricFortin](https://github.com/EricFortin))
- Windows eol [\#182](https://github.com/GoogleCloudPlatform/agones/pull/182) ([fooock](https://github.com/fooock))
- Missed a couple of small things in last PR [\#175](https://github.com/GoogleCloudPlatform/agones/pull/175) ([markmandel](https://github.com/markmandel))
- Centralise utilities for testing controllers [\#172](https://github.com/GoogleCloudPlatform/agones/pull/172) ([markmandel](https://github.com/markmandel))
- Generate the install.yaml from `helm template` [\#171](https://github.com/GoogleCloudPlatform/agones/pull/171) ([markmandel](https://github.com/markmandel))
- Integrated Helm into the `build` and development system [\#170](https://github.com/GoogleCloudPlatform/agones/pull/170) ([markmandel](https://github.com/markmandel))
- Refactor of workerqueue health testing [\#164](https://github.com/GoogleCloudPlatform/agones/pull/164) ([markmandel](https://github.com/markmandel))
- Fix some Go Report Card warnings [\#162](https://github.com/GoogleCloudPlatform/agones/pull/162) ([dvrkps](https://github.com/dvrkps))
- fix typo found by report card [\#161](https://github.com/GoogleCloudPlatform/agones/pull/161) ([Kuqd](https://github.com/Kuqd))
- Why does this project exist? [\#159](https://github.com/GoogleCloudPlatform/agones/pull/159) ([markmandel](https://github.com/markmandel))
- Add GKE Comic to explain Kubernetes to newcomers [\#158](https://github.com/GoogleCloudPlatform/agones/pull/158) ([markmandel](https://github.com/markmandel))
- Adding a Go Report Card [\#157](https://github.com/GoogleCloudPlatform/agones/pull/157) ([markmandel](https://github.com/markmandel))
- Documentation on the CI build system. [\#152](https://github.com/GoogleCloudPlatform/agones/pull/152) ([markmandel](https://github.com/markmandel))
- Helm integration [\#149](https://github.com/GoogleCloudPlatform/agones/pull/149) ([fooock](https://github.com/fooock))
- Minor rewording [\#148](https://github.com/GoogleCloudPlatform/agones/pull/148) ([bransorem](https://github.com/bransorem))
- Move GameServer validation to a ValidatingAdmissionWebhook [\#147](https://github.com/GoogleCloudPlatform/agones/pull/147) ([markmandel](https://github.com/markmandel))
- Update create\_gameserver.md [\#143](https://github.com/GoogleCloudPlatform/agones/pull/143) ([royingantaginting](https://github.com/royingantaginting))
- Fixed spelling issues in gameserver\_spec.md [\#141](https://github.com/GoogleCloudPlatform/agones/pull/141) ([mattva01](https://github.com/mattva01))
- Handle stop signal in the SDK Server [\#140](https://github.com/GoogleCloudPlatform/agones/pull/140) ([markmandel](https://github.com/markmandel))
- go vet: 3 warnings, 2 of them are easy. [\#139](https://github.com/GoogleCloudPlatform/agones/pull/139) ([Deleplace](https://github.com/Deleplace))
- Update Go version to 1.10 [\#137](https://github.com/GoogleCloudPlatform/agones/pull/137) ([markmandel](https://github.com/markmandel))
- Cleanup of grpc go generation code [\#136](https://github.com/GoogleCloudPlatform/agones/pull/136) ([markmandel](https://github.com/markmandel))
- Update base version to 0.2 [\#133](https://github.com/GoogleCloudPlatform/agones/pull/133) ([markmandel](https://github.com/markmandel))
- Centralise the canonical import paths and more package docs [\#130](https://github.com/GoogleCloudPlatform/agones/pull/130) ([markmandel](https://github.com/markmandel))

## [v0.1](https://github.com/GoogleCloudPlatform/agones/tree/v0.1) (2018-03-06)

[Full Changelog](https://github.com/GoogleCloudPlatform/agones/compare/20f6ab798a49e3629d5f6651201504ff49ea251a...v0.1)

**Implemented enhancements:**

- The local mode of the agon sidecar listen to localhost only [\#62](https://github.com/GoogleCloudPlatform/agones/issues/62)
- Record Events for GameServer State Changes [\#32](https://github.com/GoogleCloudPlatform/agones/issues/32)
- Use a single install.yaml to install Agon [\#17](https://github.com/GoogleCloudPlatform/agones/issues/17)
- SDK + Sidecar implementation [\#16](https://github.com/GoogleCloudPlatform/agones/issues/16)
- Game Server health checking [\#15](https://github.com/GoogleCloudPlatform/agones/issues/15)
- Dynamic Port Allocation on Game Servers [\#14](https://github.com/GoogleCloudPlatform/agones/issues/14)
- Sidecar needs a healthcheck [\#12](https://github.com/GoogleCloudPlatform/agones/issues/12)
- Health Check for the Controller [\#11](https://github.com/GoogleCloudPlatform/agones/issues/11)
- GameServer definition validation [\#10](https://github.com/GoogleCloudPlatform/agones/issues/10)
- Default RestartPolicy should be Never on the GameServer container [\#9](https://github.com/GoogleCloudPlatform/agones/issues/9)
- Mac & Windows binaries for local development [\#8](https://github.com/GoogleCloudPlatform/agones/issues/8)
- `gcloud docker --authorize` make target and push targets [\#5](https://github.com/GoogleCloudPlatform/agones/issues/5)
- Do-release target to automate releases [\#121](https://github.com/GoogleCloudPlatform/agones/pull/121) ([markmandel](https://github.com/markmandel))
- Zip archive of sdk server server binaries for release [\#118](https://github.com/GoogleCloudPlatform/agones/pull/118) ([markmandel](https://github.com/markmandel))
- add hostPort and container validations to webhook [\#106](https://github.com/GoogleCloudPlatform/agones/pull/106) ([Kuqd](https://github.com/Kuqd))
- MutatingWebHookConfiguration for GameServer creation & Validation. [\#95](https://github.com/GoogleCloudPlatform/agones/pull/95) ([markmandel](https://github.com/markmandel))
- Address flag for the sidecar [\#73](https://github.com/GoogleCloudPlatform/agones/pull/73) ([markmandel](https://github.com/markmandel))
- Allow extra args to be passed into minikube-shell [\#71](https://github.com/GoogleCloudPlatform/agones/pull/71) ([markmandel](https://github.com/markmandel))
- Implementation of Health Checking [\#69](https://github.com/GoogleCloudPlatform/agones/pull/69) ([markmandel](https://github.com/markmandel))
- Develop and Build on Windows \(WSL\) with Minikube [\#59](https://github.com/GoogleCloudPlatform/agones/pull/59) ([markmandel](https://github.com/markmandel))
- Recording GameServers Kubernetes Events [\#56](https://github.com/GoogleCloudPlatform/agones/pull/56) ([markmandel](https://github.com/markmandel))
- Add health check for gameserver-sidecar. [\#44](https://github.com/GoogleCloudPlatform/agones/pull/44) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Dynamic Port Allocation for GameServers [\#41](https://github.com/GoogleCloudPlatform/agones/pull/41) ([markmandel](https://github.com/markmandel))
- Finalizer for GameServer until backing Pods are Terminated [\#40](https://github.com/GoogleCloudPlatform/agones/pull/40) ([markmandel](https://github.com/markmandel))
- Continuous Integration with Container Builder [\#38](https://github.com/GoogleCloudPlatform/agones/pull/38) ([markmandel](https://github.com/markmandel))
- Windows and OSX builds of the sidecar [\#36](https://github.com/GoogleCloudPlatform/agones/pull/36) ([markmandel](https://github.com/markmandel))
- C++ SDK implementation, example and doc [\#35](https://github.com/GoogleCloudPlatform/agones/pull/35) ([markmandel](https://github.com/markmandel))
- Use a sha256 of Dockerfile for build-image [\#25](https://github.com/GoogleCloudPlatform/agones/pull/25) ([markmandel](https://github.com/markmandel))
- Utilises Xonotic.org to build and run an actual game on Agon. [\#23](https://github.com/GoogleCloudPlatform/agones/pull/23) ([markmandel](https://github.com/markmandel))
- Go SDK for integration with Game Servers. [\#20](https://github.com/GoogleCloudPlatform/agones/pull/20) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- `make gcloud-auth-docker` fails on Windows [\#49](https://github.com/GoogleCloudPlatform/agones/issues/49)
- Convert `ENTRYPOINT foo` to `ENTRYPOINT \["/path/foo"\]` [\#39](https://github.com/GoogleCloudPlatform/agones/issues/39)
- Handle SIGTERM in Controller [\#33](https://github.com/GoogleCloudPlatform/agones/issues/33)
- Gopkg.toml should use tags not branches for k8s.io dependencies [\#1](https://github.com/GoogleCloudPlatform/agones/issues/1)
- fix liveness probe in the install.yaml [\#119](https://github.com/GoogleCloudPlatform/agones/pull/119) ([Kuqd](https://github.com/Kuqd))
- Make Port Allocator idempotent for GameServers and Node events [\#117](https://github.com/GoogleCloudPlatform/agones/pull/117) ([markmandel](https://github.com/markmandel))
- DeleteFunc could recieve a DeletedFinalStateUnknown [\#113](https://github.com/GoogleCloudPlatform/agones/pull/113) ([markmandel](https://github.com/markmandel))
- Goimports wasn't running on CRD generation [\#99](https://github.com/GoogleCloudPlatform/agones/pull/99) ([markmandel](https://github.com/markmandel))
- Fix a bug in HandleError [\#67](https://github.com/GoogleCloudPlatform/agones/pull/67) ([markmandel](https://github.com/markmandel))
- Minikube targts: make sure they are on the agon minikube profile [\#66](https://github.com/GoogleCloudPlatform/agones/pull/66) ([markmandel](https://github.com/markmandel))
- Header insert on gRPC code gen touched too many files [\#58](https://github.com/GoogleCloudPlatform/agones/pull/58) ([markmandel](https://github.com/markmandel))
- Fix for health check stability issues [\#55](https://github.com/GoogleCloudPlatform/agones/pull/55) ([markmandel](https://github.com/markmandel))
- `make gcloud-auth-docker` works on Windows [\#50](https://github.com/GoogleCloudPlatform/agones/pull/50) ([markmandel](https://github.com/markmandel))
- Use the preferred ENTRYPOINT format [\#43](https://github.com/GoogleCloudPlatform/agones/pull/43) ([markmandel](https://github.com/markmandel))
- Update Kubernetes dependencies to release branch [\#24](https://github.com/GoogleCloudPlatform/agones/pull/24) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Switch to RBAC [\#57](https://github.com/GoogleCloudPlatform/agones/issues/57)
- Upgrade to Go 1.9.4 [\#81](https://github.com/GoogleCloudPlatform/agones/pull/81) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- `make do-release` target [\#115](https://github.com/GoogleCloudPlatform/agones/issues/115)
- Creating a Kubernetes Cluster quickstart [\#93](https://github.com/GoogleCloudPlatform/agones/issues/93)
- Namespace for Agones infrastructure [\#89](https://github.com/GoogleCloudPlatform/agones/issues/89)
- Health check should be moved out of `gameservers/controller.go` [\#88](https://github.com/GoogleCloudPlatform/agones/issues/88)
- Add archiving the sdk-server binaries into gcs into the cloudbuild.yaml [\#87](https://github.com/GoogleCloudPlatform/agones/issues/87)
- Upgrade to Go 1.9.3 [\#63](https://github.com/GoogleCloudPlatform/agones/issues/63)
- Building Agon on Windows [\#47](https://github.com/GoogleCloudPlatform/agones/issues/47)
- Building Agones on macOS [\#46](https://github.com/GoogleCloudPlatform/agones/issues/46)
- Write documentation for creating a GameServer [\#45](https://github.com/GoogleCloudPlatform/agones/issues/45)
- Agon should work on Minikube [\#30](https://github.com/GoogleCloudPlatform/agones/issues/30)
- Remove the entrypoint from the build-image [\#28](https://github.com/GoogleCloudPlatform/agones/issues/28)
- Base Go Version and Docker image tag on Git commit sha [\#21](https://github.com/GoogleCloudPlatform/agones/issues/21)
- Tag agon-build with hash of the Dockerfile [\#19](https://github.com/GoogleCloudPlatform/agones/issues/19)
- Example using Xonotic [\#18](https://github.com/GoogleCloudPlatform/agones/issues/18)
- Continuous Integration [\#13](https://github.com/GoogleCloudPlatform/agones/issues/13)
- C++ SDK [\#7](https://github.com/GoogleCloudPlatform/agones/issues/7)
- Upgrade to alpine 3.7 [\#4](https://github.com/GoogleCloudPlatform/agones/issues/4)
- Make controller SchemeGroupVersion a var [\#3](https://github.com/GoogleCloudPlatform/agones/issues/3)
- Consolidate `Version` into a single constant [\#2](https://github.com/GoogleCloudPlatform/agones/issues/2)

**Merged pull requests:**

- Godoc badge! [\#131](https://github.com/GoogleCloudPlatform/agones/pull/131) ([markmandel](https://github.com/markmandel))
- add missing link to git message documentation [\#129](https://github.com/GoogleCloudPlatform/agones/pull/129) ([Kuqd](https://github.com/Kuqd))
- Minor tweak to top line description of Agones. [\#127](https://github.com/GoogleCloudPlatform/agones/pull/127) ([markmandel](https://github.com/markmandel))
- Documentation for programmatically working with Agones. [\#126](https://github.com/GoogleCloudPlatform/agones/pull/126) ([markmandel](https://github.com/markmandel))
- Extend documentation for SDKs [\#125](https://github.com/GoogleCloudPlatform/agones/pull/125) ([markmandel](https://github.com/markmandel))
- Documentation quickstart and specification gameserver [\#124](https://github.com/GoogleCloudPlatform/agones/pull/124) ([Kuqd](https://github.com/Kuqd))
- Add MutatingAdmissionWebhook requirements to README [\#123](https://github.com/GoogleCloudPlatform/agones/pull/123) ([markmandel](https://github.com/markmandel))
- Add cluster creation Quickstart. [\#122](https://github.com/GoogleCloudPlatform/agones/pull/122) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Fix typo introduced by refactor [\#120](https://github.com/GoogleCloudPlatform/agones/pull/120) ([markmandel](https://github.com/markmandel))
- Cleanup on GameServer Controller [\#114](https://github.com/GoogleCloudPlatform/agones/pull/114) ([markmandel](https://github.com/markmandel))
- Fixed some typos. [\#112](https://github.com/GoogleCloudPlatform/agones/pull/112) ([EricFortin](https://github.com/EricFortin))
- Added the source of the name to the Readme. [\#111](https://github.com/GoogleCloudPlatform/agones/pull/111) ([markmandel](https://github.com/markmandel))
- Add 'make' to minikube target commands [\#109](https://github.com/GoogleCloudPlatform/agones/pull/109) ([joeholley](https://github.com/joeholley))
- Move WaitForEstablishedCRD into central `crd` package [\#108](https://github.com/GoogleCloudPlatform/agones/pull/108) ([markmandel](https://github.com/markmandel))
- Centralise Controller Queue and Worker processing [\#107](https://github.com/GoogleCloudPlatform/agones/pull/107) ([markmandel](https://github.com/markmandel))
- Slack community! [\#105](https://github.com/GoogleCloudPlatform/agones/pull/105) ([markmandel](https://github.com/markmandel))
- Add an `source` to all log statements [\#103](https://github.com/GoogleCloudPlatform/agones/pull/103) ([markmandel](https://github.com/markmandel))
- Putting the code of conduct front of page. [\#102](https://github.com/GoogleCloudPlatform/agones/pull/102) ([markmandel](https://github.com/markmandel))
- Add CRD validation via OpenAPIv3 Schema [\#100](https://github.com/GoogleCloudPlatform/agones/pull/100) ([Kuqd](https://github.com/Kuqd))
- Use github.com/heptio/healthcheck [\#98](https://github.com/GoogleCloudPlatform/agones/pull/98) ([enocom](https://github.com/enocom))
- Adding CoC and Discuss mailing lists. [\#96](https://github.com/GoogleCloudPlatform/agones/pull/96) ([markmandel](https://github.com/markmandel))
- Standardize on LF line endings [\#92](https://github.com/GoogleCloudPlatform/agones/pull/92) ([enocom](https://github.com/enocom))
- Move Agones resources into a `agones-system` namespace. [\#91](https://github.com/GoogleCloudPlatform/agones/pull/91) ([markmandel](https://github.com/markmandel))
- Support builds on macOS [\#90](https://github.com/GoogleCloudPlatform/agones/pull/90) ([enocom](https://github.com/enocom))
- Enable RBAC [\#86](https://github.com/GoogleCloudPlatform/agones/pull/86) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Update everything to be Kubernetes 1.9+ [\#85](https://github.com/GoogleCloudPlatform/agones/pull/85) ([markmandel](https://github.com/markmandel))
- Expand on contributing documentation. [\#84](https://github.com/GoogleCloudPlatform/agones/pull/84) ([markmandel](https://github.com/markmandel))
- Remove entrypoints in makefile. [\#82](https://github.com/GoogleCloudPlatform/agones/pull/82) ([Kuqd](https://github.com/Kuqd))
- Update to client-go release 1.6 [\#80](https://github.com/GoogleCloudPlatform/agones/pull/80) ([markmandel](https://github.com/markmandel))
- Setup for social/get involved section. [\#79](https://github.com/GoogleCloudPlatform/agones/pull/79) ([markmandel](https://github.com/markmandel))
- Changing name from Agon =\> Agones. [\#78](https://github.com/GoogleCloudPlatform/agones/pull/78) ([markmandel](https://github.com/markmandel))
- Refactor to centralise controller [\#77](https://github.com/GoogleCloudPlatform/agones/pull/77) ([markmandel](https://github.com/markmandel))
- Missing link to healthchecking. [\#76](https://github.com/GoogleCloudPlatform/agones/pull/76) ([markmandel](https://github.com/markmandel))
- Upgrade to Alpine 3.7 [\#75](https://github.com/GoogleCloudPlatform/agones/pull/75) ([markmandel](https://github.com/markmandel))
- Update to Go 1.9.3 [\#74](https://github.com/GoogleCloudPlatform/agones/pull/74) ([markmandel](https://github.com/markmandel))
- Update Xonotic demo to use dynamic ports [\#72](https://github.com/GoogleCloudPlatform/agones/pull/72) ([markmandel](https://github.com/markmandel))
- Basic structure for better documentation [\#68](https://github.com/GoogleCloudPlatform/agones/pull/68) ([markmandel](https://github.com/markmandel))
- Update gke-test-cluster admin password to new minimum length 16 chars. [\#65](https://github.com/GoogleCloudPlatform/agones/pull/65) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Output the stack error as an actual array [\#61](https://github.com/GoogleCloudPlatform/agones/pull/61) ([markmandel](https://github.com/markmandel))
- Update documentation [\#53](https://github.com/GoogleCloudPlatform/agones/pull/53) ([Kuqd](https://github.com/Kuqd))
- Correct maximum parameter typo [\#52](https://github.com/GoogleCloudPlatform/agones/pull/52) ([Kuqd](https://github.com/Kuqd))
- Document building Agon on different platforms [\#51](https://github.com/GoogleCloudPlatform/agones/pull/51) ([markmandel](https://github.com/markmandel))
- Development and Deployment to Minikube [\#48](https://github.com/GoogleCloudPlatform/agones/pull/48) ([markmandel](https://github.com/markmandel))
- Fix typo for logrus gameserver field [\#42](https://github.com/GoogleCloudPlatform/agones/pull/42) ([alexandrem](https://github.com/alexandrem))
- Add health check. [\#34](https://github.com/GoogleCloudPlatform/agones/pull/34) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Guide for developing and building Agon. [\#31](https://github.com/GoogleCloudPlatform/agones/pull/31) ([markmandel](https://github.com/markmandel))
- Implement Versioning for dev and release [\#29](https://github.com/GoogleCloudPlatform/agones/pull/29) ([markmandel](https://github.com/markmandel))
- Consolidate the Version constant [\#27](https://github.com/GoogleCloudPlatform/agones/pull/27) ([markmandel](https://github.com/markmandel))
- Make targets `gcloud docker --authorize-only` and `push` [\#26](https://github.com/GoogleCloudPlatform/agones/pull/26) ([markmandel](https://github.com/markmandel))
- zinstall.yaml to install Agon. [\#22](https://github.com/GoogleCloudPlatform/agones/pull/22) ([markmandel](https://github.com/markmandel))
- Disclaimer that Agon is alpha software. [\#6](https://github.com/GoogleCloudPlatform/agones/pull/6) ([markmandel](https://github.com/markmandel))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
