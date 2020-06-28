// Copyright 2019 Google LLC All Rights Reserved.
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

const messages = require('../lib/alpha/alpha_pb');
const AlphaAgonesSDK = require('../src/alphaAgonesSDK');

describe('alphaAgonesSDK', () => {
	let agonesSDK;

	beforeEach(() => {
		agonesSDK = new AlphaAgonesSDK();
	});

	describe('playerConnect', () => {
		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(agonesSDK.alphaClient, 'playerConnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await agonesSDK.playerConnect('playerID');
			expect(agonesSDK.alphaClient.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is already connected', async () => {
			spyOn(agonesSDK.alphaClient, 'playerConnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await agonesSDK.playerConnect('playerID');
			expect(agonesSDK.alphaClient.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'playerConnect').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.playerConnect('playerID');
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.playerConnect).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('playerDisconnect', () => {
		it('calls the server and handles success when the player is connected', async () => {
			spyOn(agonesSDK.alphaClient, 'playerDisconnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await agonesSDK.playerDisconnect('playerID');
			expect(agonesSDK.alphaClient.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(agonesSDK.alphaClient, 'playerDisconnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await agonesSDK.playerDisconnect('playerID');
			expect(agonesSDK.alphaClient.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'playerDisconnect').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.playerDisconnect('playerID');
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.playerDisconnect).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setPlayerCapacity', () => {
		it('passes arguments to the server and handles success', async () => {
			spyOn(agonesSDK.alphaClient, 'setPlayerCapacity').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			let result = await agonesSDK.setPlayerCapacity(64);
			expect(result).toEqual({});
			expect(agonesSDK.alphaClient.setPlayerCapacity).toHaveBeenCalled();
			let request = agonesSDK.alphaClient.setPlayerCapacity.calls.argsFor(0)[0];
			expect(request.getCount()).toEqual(64);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'setPlayerCapacity').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.setPlayerCapacity(64);
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.setPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getPlayerCapacity', () => {
		it('calls the server and handles the response', async () => {
			spyOn(agonesSDK.alphaClient, 'getPlayerCapacity').and.callFake((request, callback) => {
				let capacity = new messages.Count();
				capacity.setCount(64);
				callback(undefined, capacity);
			});

			let capacity = await agonesSDK.getPlayerCapacity();
			expect(agonesSDK.alphaClient.getPlayerCapacity).toHaveBeenCalled();
			expect(capacity).toEqual(64);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'getPlayerCapacity').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.getPlayerCapacity();
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.getPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getPlayerCount', () => {
		it('calls the server and handles the response', async () => {
			spyOn(agonesSDK.alphaClient, 'getPlayerCount').and.callFake((request, callback) => {
				let capacity = new messages.Count();
				capacity.setCount(16);
				callback(undefined, capacity);
			});

			let capacity = await agonesSDK.getPlayerCount();
			expect(agonesSDK.alphaClient.getPlayerCount).toHaveBeenCalled();
			expect(capacity).toEqual(16);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'getPlayerCount').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.getPlayerCount();
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.getPlayerCount).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('isPlayerConnected', () => {
		it('calls the server and handles success when the player is connected', async () => {
			spyOn(agonesSDK.alphaClient, 'isPlayerConnected').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await agonesSDK.isPlayerConnected('playerID');
			expect(agonesSDK.alphaClient.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(agonesSDK.alphaClient, 'isPlayerConnected').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await agonesSDK.isPlayerConnected('playerID');
			expect(agonesSDK.alphaClient.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'isPlayerConnected').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.isPlayerConnected('playerID');
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.isPlayerConnected).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getConnectedPlayers', () => {
		it('calls the server and handles the response', async () => {
			spyOn(agonesSDK.alphaClient, 'getConnectedPlayers').and.callFake((request, callback) => {
				let connectedPlayers = new messages.PlayerIDList();
				connectedPlayers.setListList(['firstPlayerID', 'secondPlayerID']);
				callback(undefined, connectedPlayers);
			});

			let connectedPlayers = await agonesSDK.getConnectedPlayers();
			expect(agonesSDK.alphaClient.getConnectedPlayers).toHaveBeenCalled();
			expect(connectedPlayers).toEqual(['firstPlayerID', 'secondPlayerID']);
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.alphaClient, 'getConnectedPlayers').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.getConnectedPlayers();
				fail();
			} catch (error) {
				expect(agonesSDK.alphaClient.getConnectedPlayers).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});
});
