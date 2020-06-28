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

const grpc = require('@grpc/grpc-js');

const AgonesSDK = require('./agonesSDK');

const messages = require('../lib/alpha/alpha_pb');
const servicesPackageDefinition = require('../lib/alpha/alpha_grpc_pb');

class AlphaAgonesSDK extends AgonesSDK {
	constructor() {
		super();

		const services = grpc.loadPackageDefinition(servicesPackageDefinition);
        this.alphaClient = new services.agones.dev.sdk.alpha.SDK(`localhost:${this.port}`, grpc.credentials.createInsecure());
	}

	async playerConnect(playerID) {
		const request = new messages.PlayerID();
		request.setPlayerid(playerID);

		return new Promise((resolve, reject) => {
			this.alphaClient.playerConnect(request, (error, response) => {
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
			this.alphaClient.playerDisconnect(request, (error, response) => {
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
			this.alphaClient.setPlayerCapacity(request, (error, response) => {
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
			this.alphaClient.getPlayerCapacity(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	getPlayerCount() {
		const request = new messages.Empty();

		return new Promise((resolve, reject) => {
			this.alphaClient.getPlayerCount(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	isPlayerConnected(playerID) {
		const request = new messages.PlayerID();
		request.setPlayerid(playerID);

		return new Promise((resolve, reject) => {
			this.alphaClient.isPlayerConnected(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getBool());
				}
			});
		});
	}

	getConnectedPlayers() {
		const request = new messages.Empty();

		return new Promise((resolve, reject) => {
			this.alphaClient.getConnectedPlayers(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getListList());
				}
			});
		});
	}
}

module.exports = AlphaAgonesSDK;
