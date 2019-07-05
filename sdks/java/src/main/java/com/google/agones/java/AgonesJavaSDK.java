/*
 * Copyright 2019 Google LLC All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.agones.java;

import com.google.agones.java.observer.EmptyObserver;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.stub.StreamObserver;
import stable.agones.dev.sdk.SDKGrpc;
import stable.agones.dev.sdk.SDKGrpc.SDKBlockingStub;
import stable.agones.dev.sdk.SDKGrpc.SDKStub;
import stable.agones.dev.sdk.Sdk.Empty;
import stable.agones.dev.sdk.Sdk.GameServer;
import stable.agones.dev.sdk.Sdk.KeyValue;

public class AgonesJavaSDK {

  private static final String DEFAULT_HOST = "localhost";
  private static final int DEFAULT_PORT = 59357;

  private final ManagedChannel managedChannel;
  private final SDKStub stub;

  private final SDKBlockingStub blockingStub;
  private final StreamObserver<Empty> healthObserver;

  public AgonesJavaSDK(ManagedChannel managedChannel, SDKStub stub, SDKBlockingStub blockingStub,
      StreamObserver<Empty> healthObserver) {
    this.managedChannel = managedChannel;
    this.stub = stub;
    this.blockingStub = blockingStub;
    this.healthObserver = healthObserver;
  }

  public AgonesJavaSDK(String host, int port) {
    this(ManagedChannelBuilder.forAddress(host, port));
  }

  public AgonesJavaSDK() {
    this(DEFAULT_HOST, DEFAULT_PORT);
  }

  public AgonesJavaSDK(ManagedChannelBuilder<?> channelBuilder) {
    this(channelBuilder.build());
  }

  public AgonesJavaSDK(ManagedChannel managedChannel) {
    this(managedChannel, SDKGrpc.newStub(managedChannel), SDKGrpc.newBlockingStub(managedChannel));
  }

  public AgonesJavaSDK(ManagedChannel managedChannel, SDKStub stub, SDKBlockingStub blockingStub) {
    this(managedChannel, stub, blockingStub, stub.health(new EmptyObserver()));
  }

  public void allocate() {

    blockingStub.allocate(Empty.getDefaultInstance());
  }

  public void ready() {

    blockingStub.ready(Empty.getDefaultInstance());
  }

  public void shutdown() {

    blockingStub.shutdown(Empty.getDefaultInstance());
  }

  public void health() {

    healthObserver.onNext(Empty.getDefaultInstance());
  }

  public GameServer getGameServer() {

    return blockingStub.getGameServer(Empty.getDefaultInstance());
  }

  public void watchGameServer(StreamObserver<GameServer> observer) {

    stub.watchGameServer(Empty.getDefaultInstance(), observer);
  }

  public void setLabel(String key, String value) {

    blockingStub.setLabel(KeyValue.newBuilder()
        .setKey(key)
        .setValue(value)
        .build());
  }

  public void setAnnotation(String key, String value) {

    blockingStub.setAnnotation(KeyValue.newBuilder()
        .setKey(key)
        .setValue(value)
        .build());
  }
}
