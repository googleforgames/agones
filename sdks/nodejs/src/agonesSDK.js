const grpc = require('grpc');

const messages = require('../lib/sdk_pb');
const services = require('../lib/sdk_grpc_pb');

class AgonesSDK {
	constructor() {
		this.client = new services.SDKClient('localhost:59357', grpc.credentials.createInsecure());
		this.healthStream = undefined;
		this.emitters = [];
	}
	async close(){
		if (this.healthStream !== undefined){
			this.healthStream.destroy()
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
			this.healthStream = this.client.health((error) => {
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
     		if (error.code === grpc.status.CANCELLED) {  // this happens when call is cancelled
     			return; 
     		}
  		})
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
}

module.exports = AgonesSDK;
