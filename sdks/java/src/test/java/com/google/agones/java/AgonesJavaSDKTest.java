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

import static org.junit.jupiter.api.Assertions.assertEquals;

import io.grpc.ManagedChannel;
import io.grpc.stub.StreamObserver;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.junit.jupiter.MockitoExtension;
import stable.agones.dev.sdk.SDKGrpc.SDKBlockingStub;
import stable.agones.dev.sdk.SDKGrpc.SDKStub;
import stable.agones.dev.sdk.Sdk.Empty;
import stable.agones.dev.sdk.Sdk.GameServer;

@ExtendWith(MockitoExtension.class)
class AgonesJavaSDKTest {

  @Mock
  private ManagedChannel managedChannel;
  @Mock
  private SDKStub stub;
  @Mock
  private SDKBlockingStub blockingStub;
  @Mock
  private StreamObserver<Empty> streamObserver;

  @Mock
  private GameServer gameServer;

  private AgonesJavaSDK agonesJavaSDK;

  @BeforeEach
  void setUp() {
    agonesJavaSDK = new AgonesJavaSDK(managedChannel, stub, blockingStub, streamObserver);
  }

  @Test
  void testReady() {

    agonesJavaSDK.ready();
    Mockito.verify(blockingStub).ready(Mockito.any(Empty.class));
  }

  @Test
  void testAllocate() {

    agonesJavaSDK.allocate();
    Mockito.verify(blockingStub).allocate(Mockito.any(Empty.class));
  }

  @Test
  void testShutdown() {

    agonesJavaSDK.shutdown();
    Mockito.verify(blockingStub).shutdown(Mockito.any(Empty.class));
  }

  @Test
  void testHealth() {

    agonesJavaSDK.health();
    Mockito.verify(streamObserver).onNext(Mockito.any(Empty.class));
  }

  @Test
  void testGetGameServer() {

    Mockito.doReturn(gameServer).when(blockingStub).getGameServer(Mockito.any(Empty.class));
    GameServer gameServer = agonesJavaSDK.getGameServer();

    assertEquals(this.gameServer, gameServer);
  }

  @Test
  void testWatchGameServer() {

    final StreamObserver<GameServer> streamObserver = new StreamObserver<GameServer>() {
      @Override
      public void onNext(GameServer gameServer) {

      }

      @Override
      public void onError(Throwable throwable) {

      }

      @Override
      public void onCompleted() {

      }
    };

    agonesJavaSDK.watchGameServer(streamObserver);

    Mockito.verify(stub).watchGameServer(Mockito.any(Empty.class), Mockito.eq(streamObserver));
  }
}
