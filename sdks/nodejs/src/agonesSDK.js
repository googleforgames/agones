const grpc = require('grpc');

const messages = require('../lib/sdk_pb');
const services = require('../lib/sdk_grpc_pb');

class AgonesSDK {
	constructor() {
		this.client = new services.SDKClient('localhost:59357', grpc.credentials.createInsecure());
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
		let request = new messages.Empty();
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
		let request = new messages.Empty();
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
		if (!this.healthStream) {
			this.healthStream = this.client.health((error) => {
				// Ignore error as this can't be caught
			});
		}
		let request = new messages.Empty();
		this.healthStream.write(request);
	}

	async getGameServer() {
		let request = new messages.Empty();
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
		let request = new messages.Empty();
		let emitter = this.client.watchGameServer(request);
		emitter.on('data', (data) => {
			callback(data.toObject());
		});
	}

	async setLabel(key, value) {
		let request = new messages.KeyValue();
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
		let request = new messages.KeyValue();
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
}

module.exports = AgonesSDK;
