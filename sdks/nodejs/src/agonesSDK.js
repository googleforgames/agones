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

const grpc = require('grpc');

const messages = require('../lib/sdk_pb');
const services = require('../lib/sdk_grpc_pb');

class AgonesSDK {
	constructor() {
		this.client = new services.SDKClient('localhost:59357', grpc.credentials.createInsecure());
		this.healthStream = undefined;
		this.emitters = [];
	}

	async connect() {
		return new Promise((resolve, reject) => {
			this.client.waitForReady(Date.now() + 30000, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			})
		});
	}

	async close() {
		if (this.healthStream !== undefined) {
			this.healthStream.destroy();
		}
		this.emitters.forEach(emitter => emitter.call.cancel());
		this.client.close();
	}

	async allocate() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.allocate(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async ready() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.ready(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async shutdown() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.shutdown(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	health() {
		if (this.healthStream === undefined) {
			this.healthStream = this.client.health(() => {
				// Ignore error as this can't be caught
			});
		}
		const request = new messages.Empty();
		this.healthStream.write(request);
	}

	async getGameServer() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.getGameServer(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	watchGameServer(callback) {
		const request = new messages.Empty();
		const emitter = this.client.watchGameServer(request);
		emitter.on('data', (data) => {
			callback(data.toObject());
		});
		emitter.on('error', (error) => {
			if (error.code === grpc.status.CANCELLED) {
				// Capture error when call is cancelled
				return;
			}
			throw error;
		});
		this.emitters.push(emitter);
	}

	async setLabel(key, value) {
		const request = new messages.KeyValue();
		request.setKey(key);
		request.setValue(value);
		return new Promise((resolve, reject) => {
			this.client.setLabel(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async setAnnotation(key, value) {
		const request = new messages.KeyValue();
		request.setKey(key);
		request.setValue(value);
		return new Promise((resolve, reject) => {
			this.client.setAnnotation(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async reserve(duration) {
		const request = new messages.Duration();
		request.setSeconds(duration);
		return new Promise((resolve, reject) => {
			this.client.reserve(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}
}

module.exports = AgonesSDK;
