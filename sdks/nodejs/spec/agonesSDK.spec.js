const EventEmitter = require('events');

const messages = require('../lib/sdk_pb');
const AgonesSDK = require('../src/agonesSDK');

describe('agones', () => {
	let agonesSDK;

	beforeEach(() => {
		agonesSDK = new AgonesSDK();
	});

	describe('ready', () => {
		it('calls the server and handles success', async () => {
			spyOn(agonesSDK.client, 'ready').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});
			let result = await agonesSDK.ready();
			expect(agonesSDK.client.ready).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.client, 'ready').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.ready();
				fail();
			} catch (error) {
				expect(agonesSDK.client.ready).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('shutdown', () => {
		it('calls the server and handles success', async () => {
			spyOn(agonesSDK.client, 'shutdown').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			let result = await agonesSDK.shutdown();
			expect(agonesSDK.client.shutdown).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.client, 'shutdown').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.shutdown();
				fail();
			} catch (error) {
				expect(agonesSDK.client.shutdown).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('health', () => {
		it('calls the server and passes calls to stream', async () => {
			let stream = jasmine.createSpyObj('stream', ['write']);
			spyOn(agonesSDK.client, 'health').and.callFake(() => {
				return stream;
			});

			agonesSDK.health();
			expect(agonesSDK.client.health).toHaveBeenCalled();
			expect(stream.write).toHaveBeenCalled();
		});

		it('uses the same stream for subsequent calls', async () => {
			let stream = jasmine.createSpyObj('stream', ['write']);
			spyOn(agonesSDK.client, 'health').and.callFake(() => {
				return stream;
			});

			agonesSDK.health();
			agonesSDK.health();
			expect(agonesSDK.client.health.calls.count()).toEqual(1);
			expect(stream.write.calls.count()).toEqual(2);
		});

		it('calls the server and silently handles the internal error message', async () => {
			spyOn(agonesSDK.client, 'health').and.callFake((callback) => {
				callback('error', undefined);
			});
			try {
				agonesSDK.health();
				fail();
			} catch (error) {
				expect(agonesSDK.client.health).toHaveBeenCalled();
				expect(error).not.toEqual('error');
			}
		});

		it('calls the server and handles stream completing', async () => {
			let stream = jasmine.createSpyObj('stream', ['write']);
			spyOn(agonesSDK.client, 'health').and.callFake((callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
				return stream;
			});

			agonesSDK.health();
			expect(agonesSDK.client.health).toHaveBeenCalled();
		});
	});

	describe('getGameServer', () => {
		it('calls the server and handles the response', async () => {
			spyOn(agonesSDK.client, 'getGameServer').and.callFake((request, callback) => {
				let status = new messages.GameServer.Status();
				status.setState('up');
				let gameServer = new messages.GameServer();
				gameServer.setStatus(status);
				callback(undefined, gameServer);
			});

			let gameServer = await agonesSDK.getGameServer();
			expect(agonesSDK.client.getGameServer).toHaveBeenCalled();
			expect(gameServer).toBeDefined();
			expect(gameServer.status).toBeDefined();
			expect(gameServer.status.state).toEqual('up');
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.client, 'getGameServer').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.getGameServer();
				fail();
			} catch (error) {
				expect(agonesSDK.client.getGameServer).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('watchGameServer', () => {
		it('calls the server and passes events to the callback', async () => {
			let serverEmitter = new EventEmitter();
			spyOn(agonesSDK.client, 'watchGameServer').and.callFake(() => {
				return serverEmitter;
			});

			let callback = jasmine.createSpy('callback');
			agonesSDK.watchGameServer(callback);
			expect(agonesSDK.client.watchGameServer).toHaveBeenCalled();

			let status = new messages.GameServer.Status();
			status.setState('up');
			let gameServer = new messages.GameServer();
			gameServer.setStatus(status);
			serverEmitter.emit('data', gameServer);

			expect(callback).toHaveBeenCalled();
			let result = callback.calls.argsFor(0)[0];
			expect(result.status).toBeDefined();
			expect(result.status.state).toEqual('up');
		});
	});

	describe('setLabel', () => {
		it('calls the server and handles success', async () => {
			spyOn(agonesSDK.client, 'setLabel').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			let result = await agonesSDK.setLabel('key', 'value');
			expect(agonesSDK.client.setLabel).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it('passes arguments to the server', async () => {
			spyOn(agonesSDK.client, 'setLabel').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			await agonesSDK.setLabel('key', 'value');
			expect(agonesSDK.client.setLabel).toHaveBeenCalled();
			let request = agonesSDK.client.setLabel.calls.argsFor(0)[0];
			expect(request.getKey()).toEqual('key');
			expect(request.getValue()).toEqual('value');
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.client, 'setLabel').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.setLabel('key', 'value');
				fail();
			} catch (error) {
				expect(agonesSDK.client.setLabel).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});

	describe('setAnnotation', () => {
		it('calls the server and handles success', async () => {
			spyOn(agonesSDK.client, 'setAnnotation').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			let result = await agonesSDK.setAnnotation('key', 'value');
			expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it('passes arguments to the server', async () => {
			spyOn(agonesSDK.client, 'setAnnotation').and.callFake((request, callback) => {
				let result = new messages.Empty();
				callback(undefined, result);
			});

			await agonesSDK.setAnnotation('key', 'value');
			expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
			let request = agonesSDK.client.setAnnotation.calls.argsFor(0)[0];
			expect(request.getKey()).toEqual('key');
			expect(request.getValue()).toEqual('value');
		});

		it('calls the server and handles failure', async () => {
			spyOn(agonesSDK.client, 'setAnnotation').and.callFake((request, callback) => {
				callback('error', undefined);
			});
			try {
				await agonesSDK.setAnnotation('key', 'value');
				fail();
			} catch (error) {
				expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
				expect(error).toEqual('error');
			}
		});
	});
	describe('close', () => {
		it('closes the client connection when called', async () => {
			spyOn(agonesSDK.client, 'close').and.callFake(()=>{});
			await agonesSDK.close();
			expect(agonesSDK.client.close).toHaveBeenCalled();
		});
	});
});
