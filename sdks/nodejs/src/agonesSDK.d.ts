// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Alpha from './alpha';

type Seconds = number;

export declare class AgonesSDK {
    constructor();

    alpha: Alpha

    get port(): string

    connect(): Promise<void>

    close(): void

    ready(): Promise<Record<string, any>>

    allocate(): Promise<Record<string, any>>

    shutdown(): Promise<Record<string, any>>

    health(errorCallback: (error: any) => any): void

    getGameServer(): Promise<Record<string, any>>

    watchGameServer(callback: (data: Record<string, any>) => any, errorCallback: (error: any) => any): void

    setLabel(key: string, value: string): Promise<Record<string, any>>

    setAnnotation(key: string, value: string): Promise<Record<string, any>>

    reserve(duration: Seconds): Promise<Record<string, any>>
}

export default AgonesSDK
