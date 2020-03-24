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

const AgonesSDK = require('@google-cloud/agones-sdk');
const util = require('util');

const sleep = util.promisify(setTimeout);

const agonesSDK = new AgonesSDK();

const DEFAULT_TIMEOUT = 60;
const MAX_TIMEOUT = 2147483;

const connect = async (timeout) => {
	let lifetimeInterval;
	let healthInterval;
	
	try {
		console.log(`Connecting to the SDK server...`);
		await agonesSDK.connect();
		console.log('...connected to SDK server');

		let connectTime = Date.now();
		lifetimeInterval = setInterval(() => {
			let lifetimeSeconds = Math.floor((Date.now() - connectTime) / 1000);
			console.log(`Running for ${lifetimeSeconds} seconds`);
		}, 10000);
		healthInterval = setInterval(() => {
			agonesSDK.health();
			console.log('Health ping sent');
		}, 20000);
		agonesSDK.watchGameServer((result) => {
			console.log(`GameServer Update:\n\tname: ${result.objectMeta.name}\n\tstate: ${result.status.state}\n\tlabels: ${result.objectMeta.labelsMap.join(' & ')}\n\tannotations: ${result.objectMeta.annotationsMap.join(' & ')}`);
		});

		await sleep(10000);
		console.log('Setting a label');
		await agonesSDK.setLabel('test-label', 'test-value');

		await sleep(10000);
		console.log('Setting an annotation');
		await agonesSDK.setAnnotation('test-annotation', 'test value');

		await sleep(10000);
		console.log('Marking server as ready...');
		await agonesSDK.ready();
		
		await sleep(10000);
		console.log('Allocating');
		await agonesSDK.allocate();

		await sleep(10000);
		console.log('Reserving for 10 seconds');
		await agonesSDK.reserve(10);
		await sleep(20000);

		if (timeout === 0) {
			do {
				await sleep(MAX_TIMEOUT);
			} while (true);
		}

		console.log(`Shutting down after timeout of ${timeout} seconds...`);
		await sleep(timeout * 1000);
		console.log('Shutting down...');
		agonesSDK.shutdown();

		await sleep(10000);
		console.log('Closing connection to SDK server');
		clearInterval(healthInterval);
		agonesSDK.close();

		await sleep(10000);
		console.log('Exiting');
		clearInterval(lifetimeInterval);

		// Some parts of the SDK are still using the event loop so must exit manually until fixed
		process.exit(0);
	} catch (error) {
		console.error(error);
		clearInterval(healthInterval);
		clearInterval(lifetimeInterval);
	}
};

let args = process.argv.slice(2);
let timeout = DEFAULT_TIMEOUT;

for (let arg of args) {
	let [argName, argValue] = arg.split('=');
	if (argName === '--help') {
		console.log(`Example to call each SDK feature in turn. Once complete will call shutdown and close after a default timeout of ${DEFAULT_TIMEOUT} seconds.\n\nOptions:\n\t--timeout=...\t\tshutdown timeout in seconds. Use 0 to never shut down`);
		return;
	}
	if (argName === '--timeout') {
		timeout = Number(argValue);
		if (Number.isNaN(timeout)) {
			timeout = DEFAULT_TIMEOUT;
		} else if (timeout < 0 || timeout > MAX_TIMEOUT) {
			timeout = 0;
		}
		if (timeout === 0) {
			console.log(`Using shutdown timeout of never`);
		} else {
			console.log(`Using shutdown timeout of ${timeout} seconds`);
		}
	}
}

connect(timeout);
