// Copyright 2023 Google LLC All Rights Reserved.
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

type PlayerId = string;

declare class Alpha {
	playerConnect(playerID: PlayerId): Promise<boolean>

	playerDisconnect(playerID: PlayerId): Promise<boolean>

	setPlayerCapacity(capacity: number): Promise<Record<string, any>>

	getPlayerCapacity(): Promise<number>

	getPlayerCount(): Promise<number>

	isPlayerConnected(playerID: PlayerId): Promise<boolean>

	getConnectedPlayers(): Promise<PlayerId[]>

	getCounterCount(key: string): Promise<number>

	incrementCounter(key: string, amount: number): Promise<bool>

	decrementCounter(key: string, amount: number): Promise<bool>

	setCounterCount(key: string, amount: int64): Promise<bool>

	getCounterCapacity(key: string): Promise<bool>

	setCounterCapacity(key: string, amount: number): Promise<bool>

	getListCapacity(key: string): Promise<number>

	setListCapacity(key: string, amount: int64): Promise<bool>

	listContains(key: string, value: string): Promise<bool>

	getListLength(key: string): Promise<number>

	getListValues(key: string): Promise<string[]>

	appendListValue(key: string, value: string): Promise<bool>

	deleteListValue(key:string , value: string): Promise<bool>
}

export default Alpha;
