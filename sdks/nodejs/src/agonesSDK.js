const grpc = require('grpc');

const messages = require('../lib/sdk_pb');
const services = require('../lib/sdk_grpc_pb');

class AgonesSDK {
	constructor() {
		console.log('constructor called')
		this._client = new services.SDKClient('localhost:59357', grpc.credentials.createInsecure());
		this._healthStream = undefined;
		this._emitters = [];
	}
	async close(){
		if (this.healthStream !== undefined){
			this._healthStream.destroy()
		}
		this._emitters.forEach(emitter => emitter.call.cancel());
		this._client.close();
	}
	async ready() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this._client.ready(request, (error, response) => {
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
			this._client.shutdown(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}
	health() {
		if (this._healthStream === undefined) {
			this._healthStream = this._client.health((error) => {
				// Ignore error as this can't be caught
			});
		}
		this._healthStream.write(new messages.Empty());
	}
	async getGameServer() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this._client.getGameServer(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}
	watchGameServer(callback) {	
		const emitter = this._client.watchGameServer(new messages.Empty());
		emitter.on('data', (data) => {
			callback(data.toObject());
		});
		emitter.on('error', (error) => {
     		if (error.code === grpc.status.CANCELLED) {  // this happens when call is cancelled
     			return; 
     		}
  		})
		this._emitters.push(emitter);
	}
	async setLabel(key, value) {
		const request = new messages.KeyValue();
		request.setKey(key);
		request.setValue(value);
		return new Promise((resolve, reject) => {
			this._client.setLabel(request, (error, response) => {
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
			this._client.setAnnotation(request, (error, response) => {
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
