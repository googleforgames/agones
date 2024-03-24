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

const grpc = require('@grpc/grpc-js');

const messages = require('../lib/alpha/alpha_pb');
const Alpha = require('../src/alpha');

describe('Alpha', () => {
	let alpha;

	beforeEach(() => {
		const address = 'localhost:9357';
		const credentials = grpc.credentials.createInsecure();
		alpha = new Alpha(address, credentials);
	});

	describe('playerConnect', () => {
		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(alpha.client, 'playerConnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await alpha.playerConnect('playerID');
			expect(alpha.client.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is already connected', async () => {
			spyOn(alpha.client, 'playerConnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await alpha.playerConnect('playerID');
			expect(alpha.client.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'playerConnect').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.playerConnect('playerID');
				fail();
			} catch (error) {
				expect(alpha.client.playerConnect).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('playerDisconnect', () => {
		it('calls the server and handles success when the player is connected', async () => {
			spyOn(alpha.client, 'playerDisconnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await alpha.playerDisconnect('playerID');
			expect(alpha.client.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(alpha.client, 'playerDisconnect').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await alpha.playerDisconnect('playerID');
			expect(alpha.client.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'playerDisconnect').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.playerDisconnect('playerID');
				fail();
			} catch (error) {
				expect(alpha.client.playerDisconnect).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setPlayerCapacity', () => {
		it('passes arguments to the server and handles success', async () => {
			spyOn(alpha.client, 'setPlayerCapacity').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			let result = await alpha.setPlayerCapacity(64);
			expect(result).toEqual({});
			expect(alpha.client.setPlayerCapacity).toHaveBeenCalled();
			let request = alpha.client.setPlayerCapacity.calls.argsFor(0)[0];
			expect(request.getCount()).toEqual(64);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'setPlayerCapacity').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.setPlayerCapacity(64);
				fail();
			} catch (error) {
				expect(alpha.client.setPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getPlayerCapacity', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getPlayerCapacity').and.callFake((request, callback) => {
				let capacity = new messages.Count();
				capacity.setCount(64);
				callback(undefined, capacity);
			});

			let capacity = await alpha.getPlayerCapacity();
			expect(alpha.client.getPlayerCapacity).toHaveBeenCalled();
			expect(capacity).toEqual(64);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getPlayerCapacity').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getPlayerCapacity();
				fail();
			} catch (error) {
				expect(alpha.client.getPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getPlayerCount', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getPlayerCount').and.callFake((request, callback) => {
				let capacity = new messages.Count();
				capacity.setCount(16);
				callback(undefined, capacity);
			});

			let capacity = await alpha.getPlayerCount();
			expect(alpha.client.getPlayerCount).toHaveBeenCalled();
			expect(capacity).toEqual(16);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getPlayerCount').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getPlayerCount();
				fail();
			} catch (error) {
				expect(alpha.client.getPlayerCount).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('isPlayerConnected', () => {
		it('calls the server and handles success when the player is connected', async () => {
			spyOn(alpha.client, 'isPlayerConnected').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(true);
				callback(undefined, result);
			});

			let result = await alpha.isPlayerConnected('playerID');
			expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it('calls the server and handles success when the player is not connected', async () => {
			spyOn(alpha.client, 'isPlayerConnected').and.callFake((request, callback) => {
				let result = new messages.Bool();
				result.setBool(false);
				callback(undefined, result);
			});

			let result = await alpha.isPlayerConnected('playerID');
			expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'isPlayerConnected').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.isPlayerConnected('playerID');
				fail();
			} catch (error) {
				expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getConnectedPlayers', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getConnectedPlayers').and.callFake((request, callback) => {
				let connectedPlayers = new messages.PlayerIDList();
				connectedPlayers.setListList(['firstPlayerID', 'secondPlayerID']);
				callback(undefined, connectedPlayers);
			});

			let connectedPlayers = await alpha.getConnectedPlayers();
			expect(alpha.client.getConnectedPlayers).toHaveBeenCalled();
			expect(connectedPlayers).toEqual(['firstPlayerID', 'secondPlayerID']);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getConnectedPlayers').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getConnectedPlayers();
				fail();
			} catch (error) {
				expect(alpha.client.getConnectedPlayers).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getCounterCount', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				counter.setCount(10);
				callback(undefined, counter);
			});

			let count = await alpha.getCounterCount('key');
			expect(alpha.client.getCounter).toHaveBeenCalled();
			expect(count).toEqual(10);
			let request = alpha.client.getCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getCounterCount('key');
				fail();
			} catch (error) {
				expect(alpha.client.getCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('incrementCounter', () => {
		it('calls the server and handles the response while under capacity', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				callback(undefined, counter);
			});

			let response = await alpha.incrementCounter('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(true);
			let request = alpha.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
			expect(request.getCountdiff()).toEqual(5);
		});

		it('calls the server and handles the response while over capacity', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('OUT_OF_RANGE', undefined);
			});

			let response = await alpha.incrementCounter('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.incrementCounter('key', 5);
				fail();
			} catch (error) {
				expect(alpha.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('decrementCounter', () => {
		it('calls the server and handles the response while above zero', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				callback(undefined, counter);
			});

			let response = await alpha.decrementCounter('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(true);
			let request = alpha.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
			expect(request.getCountdiff()).toEqual(-5);
		});

		it('calls the server and handles the response while over capacity', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('OUT_OF_RANGE', undefined);
			});

			let response = await alpha.decrementCounter('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.decrementCounter('key', 5);
				fail();
			} catch (error) {
				expect(alpha.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setCounterCount', () => {
		it('calls the server and handles the response while in range', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				callback(undefined, counter);
			});

			let response = await alpha.setCounterCount('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(true);
			let request = alpha.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
			expect(request.getCount().getValue()).toEqual(5);
		});

		it('calls the server and handles the response out of range', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('OUT_OF_RANGE', undefined);
			});

			let response = await alpha.setCounterCount('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.setCounterCount('key', 5);
				fail();
			} catch (error) {
				expect(alpha.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getCounterCapacity', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				counter.setCapacity(10);
				callback(undefined, counter);
			});

			let capacity = await alpha.getCounterCapacity('key');
			expect(alpha.client.getCounter).toHaveBeenCalled();
			expect(capacity).toEqual(10);
			let request = alpha.client.getCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getCounterCapacity('key');
				fail();
			} catch (error) {
				expect(alpha.client.getCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setCounterCapacity', () => {
		it('calls the server and handles the response while in range', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				let counter = new messages.Counter();
				callback(undefined, counter);
			});

			let response = await alpha.setCounterCapacity('key', 5);
			expect(alpha.client.updateCounter).toHaveBeenCalled();
			expect(response).toBeUndefined();
			let request = alpha.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getCounterupdaterequest().getName()).toEqual('key');
			expect(request.getCounterupdaterequest().getCapacity().getValue()).toEqual(5);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'updateCounter').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.setCounterCapacity('key', 5);
				fail();
			} catch (error) {
				expect(alpha.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getListCapacity', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				let list = new messages.List();
				list.setCapacity(10);
				callback(undefined, list);
			});

			let capacity = await alpha.getListCapacity('key');
			expect(alpha.client.getList).toHaveBeenCalled();
			expect(capacity).toEqual(10);
			let request = alpha.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getListCapacity('key');
				fail();
			} catch (error) {
				expect(alpha.client.getList).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setListCapacity', () => {
		it('calls the server and handles the response while in range', async () => {
			spyOn(alpha.client, 'updateList').and.callFake((request, callback) => {
				let list = new messages.List();
				callback(undefined, list);
			});

			let response = await alpha.setListCapacity('key', 5);
			expect(alpha.client.updateList).toHaveBeenCalled();
			expect(response).toBeUndefined();
			let request = alpha.client.updateList.calls.argsFor(0)[0];
			expect(request.getList().getName()).toEqual('key');
			expect(request.getList().getCapacity()).toEqual(5);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'updateList').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.setListCapacity('key', 5);
				fail();
			} catch (error) {
				expect(alpha.client.updateList).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('listContains', () => {
		it('calls the server and handles the response when the value is contained', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				let list = new messages.List();
				list.setValuesList(['firstValue', 'secondValue']);
				callback(undefined, list);
			});

			let contains = await alpha.listContains('key', 'secondValue');
			expect(alpha.client.getList).toHaveBeenCalled();
			expect(contains).toEqual(true);
			let request = alpha.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles the response when the value is not contained', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				let list = new messages.List();
				list.setValuesList(['firstValue', 'secondValue']);
				callback(undefined, list);
			});

			let contains = await alpha.listContains('key', 'thirdValue');
			expect(alpha.client.getList).toHaveBeenCalled();
			expect(contains).toEqual(false);
			let request = alpha.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.listContains('key', 'value');
				fail();
			} catch (error) {
				expect(alpha.client.getList).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getListLength', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				let list = new messages.List();
				list.setValuesList(['firstValue', 'secondValue']);
				callback(undefined, list);
			});

			let length = await alpha.getListLength('key');
			expect(alpha.client.getList).toHaveBeenCalled();
			expect(length).toEqual(2);
			let request = alpha.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getListLength('key');
				fail();
			} catch (error) {
				expect(alpha.client.getList).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('getListValues', () => {
		it('calls the server and handles the response', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				let list = new messages.List();
				list.setValuesList(['firstValue', 'secondValue']);
				callback(undefined, list);
			});

			let values = await alpha.getListValues('key');
			expect(alpha.client.getList).toHaveBeenCalled();
			expect(values).toEqual(['firstValue', 'secondValue']);
			let request = alpha.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'getList').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.getListValues('key');
				fail();
			} catch (error) {
				expect(alpha.client.getList).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('appendListValue', () => {
		it('calls the server and handles the response when the value is appended', async () => {
			spyOn(alpha.client, 'addListValue').and.callFake((request, callback) => {
				let list = new messages.List();
				callback(undefined, list);
			});

			let response = await alpha.appendListValue('key', 'value');
			expect(alpha.client.addListValue).toHaveBeenCalled();
			expect(response).toEqual(true);
			let request = alpha.client.addListValue.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
			expect(request.getValue()).toEqual('value');
		});

		it('calls the server and handles the response when capacity is reached', async () => {
			spyOn(alpha.client, 'addListValue').and.callFake((request, callback) => {
				callback('OUT_OF_RANGE', undefined);
			});

			let response = await alpha.appendListValue('key', 'value');
			expect(alpha.client.addListValue).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles the reponse when the value already exists', async () => {
			spyOn(alpha.client, 'addListValue').and.callFake((request, callback) => {
				callback('ALREADY_EXISTS', undefined);
			});

			let response = await alpha.appendListValue('key', 'value');
			expect(alpha.client.addListValue).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'addListValue').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.appendListValue('key', 'value');
				fail();
			} catch (error) {
				expect(alpha.client.addListValue).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('deleteListValue', () => {
		it('calls the server and handles the response when the value is deleted', async () => {
			spyOn(alpha.client, 'removeListValue').and.callFake((request, callback) => {
				let list = new messages.List();
				callback(undefined, list);
			});

			let response = await alpha.deleteListValue('key', 'value');
			expect(alpha.client.removeListValue).toHaveBeenCalled();
			expect(response).toEqual(true);
			let request = alpha.client.removeListValue.calls.argsFor(0)[0];
			expect(request.getName()).toEqual('key');
			expect(request.getValue()).toEqual('value');
		});

		it('calls the server and handles the response when the value does not exist', async () => {
			spyOn(alpha.client, 'removeListValue').and.callFake((request, callback) => {
				callback('NOT_FOUND', undefined);
			});

			let response = await alpha.deleteListValue('key', 'value');
			expect(alpha.client.removeListValue).toHaveBeenCalled();
			expect(response).toEqual(false);
		});

		it('calls the server and handles failure', async () => {
			spyOn(alpha.client, 'removeListValue').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await alpha.deleteListValue('key', 'value');
				fail();
			} catch (error) {
				expect(alpha.client.removeListValue).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});
});
