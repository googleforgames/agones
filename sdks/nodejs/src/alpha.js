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
const jspbWrappers = require('google-protobuf/google/protobuf/wrappers_pb');

const messages = require('../lib/alpha/alpha_pb');
const servicesPackageDefinition = require('../lib/alpha/alpha_grpc_pb');

class Alpha {
	constructor(address, credentials) {
		const services = grpc.loadPackageDefinition(servicesPackageDefinition);
		this.client = new services.agones.dev.sdk.alpha.SDK(address, credentials);
	}

	async playerConnect(playerID) {
		const request = new messages.PlayerID();
		request.setPlayerid(playerID);

		return new Promise((resolve, reject) => {
			this.client.playerConnect(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getBool());
				}
			});
		});
	}

	async playerDisconnect(playerID) {
		const request = new messages.PlayerID();
		request.setPlayerid(playerID);

		return new Promise((resolve, reject) => {
			this.client.playerDisconnect(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getBool());
				}
			});
		});
	}

	async setPlayerCapacity(capacity) {
		const request = new messages.Count();
		request.setCount(capacity);

		return new Promise((resolve, reject) => {
			this.client.setPlayerCapacity(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async getPlayerCapacity() {
		const request = new messages.Empty();

		return new Promise((resolve, reject) => {
			this.client.getPlayerCapacity(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	async getPlayerCount() {
		const request = new messages.Empty();

		return new Promise((resolve, reject) => {
			this.client.getPlayerCount(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	async isPlayerConnected(playerID) {
		const request = new messages.PlayerID();
		request.setPlayerid(playerID);

		return new Promise((resolve, reject) => {
			this.client.isPlayerConnected(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getBool());
				}
			});
		});
	}

	async getConnectedPlayers() {
		const request = new messages.Empty();

		return new Promise((resolve, reject) => {
			this.client.getConnectedPlayers(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getListList());
				}
			});
		});
	}

	async getCounterCount(key) {
		const request = new messages.GetCounterRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getCounter(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	async incrementCounter(key, amount) {
		const request = new messages.CounterUpdateRequest();
		request.setName(key);
		request.setCountdiff(amount);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async decrementCounter(key, amount) {
		const request = new messages.CounterUpdateRequest();
		request.setName(key);
		request.setCountdiff(-amount);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async setCounterCount(key, amount) {
		let count = new jspbWrappers.Int64Value();
		count.setValue(amount);
		const request = new messages.CounterUpdateRequest();
		request.setName(key);
		request.setCount(count);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async getCounterCapacity(key) {
		const request = new messages.GetCounterRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getCounter(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCapacity());
				}
			});
		});
	}

	async setCounterCapacity(key, amount) {
		const capacity = new jspbWrappers.Int64Value();
		capacity.setValue(amount);
		const updateRequest = new messages.CounterUpdateRequest();
		updateRequest.setName(key);
		updateRequest.setCapacity(capacity);
		const request = new messages.UpdateCounterRequest();
		request.setCounterupdaterequest(updateRequest);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async getListCapacity(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCapacity());
				}
			});
		});
	}

	async setListCapacity(key, amount) {
		const list = new messages.List();
		list.setName(key);
		list.setCapacity(amount);
		const request = new messages.UpdateListRequest();
		request.setList(list);

		return new Promise((resolve, reject) => {
			this.client.updateList(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async listContains(key, value) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList().indexOf(value) >= 0);
				}
			});
		});
	}

	async getListLength(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList().length);
				}
			});
		});
	}

	async getListValues(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList());
				}
			});
		});
	}

	async appendListValue(key, value) {
		const request = new messages.AddListValueRequest();
		request.setName(key);
		request.setValue(value);

		return new Promise((resolve, reject) => {
			this.client.addListValue(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async deleteListValue(key, value) {
		const request = new messages.RemoveListValueRequest();
		request.setName(key);
		request.setValue(value);

		return new Promise((resolve, reject) => {
			this.client.removeListValue(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}
}

module.exports = Alpha;
