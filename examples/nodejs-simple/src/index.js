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

const AgonesSDK = require('@googleforgames/agones');

const agonesSDK = new AgonesSDK();

const connect = async () => {
	agonesSDK.watchGameServer((result) => {
		console.log('GameServer Update:\n\tname:', result.objectMeta.name, '\n\tstate:', result.status.state);
	});
	let healthInterval = setInterval(() => {
		agonesSDK.health();
		console.log('Health ping sent');
	}, 20000);

	try {
		console.log('node.js Game Server has started!');

		console.log('Setting a label');
		await agonesSDK.setLabel('test-label', 'test-value');
		console.log('Setting an annotation');
		await agonesSDK.setAnnotation('test-annotation', 'test value');

		console.log('Marking server as ready...');
		await agonesSDK.ready();
		console.log('...marked Ready');

		let count = 0;
		setInterval(() => {
			count = count + 10;
			console.log('Running for', count, 'seconds!');
		}, 10000);
		setTimeout(async () => {
			console.log('Allocating');
			await agonesSDK.allocate();
		}, 15000);
		setTimeout(async () => {
			console.log('Reserving for 20 seconds');
			await agonesSDK.reserve(20);
		}, 25000);
		setTimeout(() => {
			console.log('Shutting down after 60 seconds...');
			agonesSDK.shutdown();
			console.log('...marked for Shutdown');
		}, 60000);
		setTimeout(() => {
			agonesSDK.close();
		}, 90000);
		setTimeout(() => {
			process.exit(0);
		}, 100000);
	} catch (error) {
		console.error(error);
		clearInterval(healthInterval);
	}
};

connect();
