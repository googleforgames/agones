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
const agonesSDK = new AgonesSDK();

const connect = async () => {
	let UID = '';
	try {
		console.log("attempting to connect");
		await agonesSDK.connect();
		console.log("connected!");
		let once = true;
		agonesSDK.watchGameServer((result) => {
			console.log('watch', result);
			UID = result.objectMeta.uid.toString();
			if (once) {
				console.log('Setting annotation ', UID);
				agonesSDK.setAnnotation('annotation', UID);
				once = false;
			}
		}, (error) => {
			console.error('Watch ERROR', error);
		});
		await agonesSDK.ready();

		let result = await agonesSDK.getGameServer();
		await agonesSDK.setLabel('label', result.objectMeta.creationTimestamp.toString());
		console.log('gameServer', result);
		console.log('creation Timestamp', result.objectMeta.creationTimestamp);
		result = await agonesSDK.health();
		console.log('health', result);

		await agonesSDK.reserve(5);

		setTimeout(() => {
			console.log('send allocate request');
			agonesSDK.allocate();
		}, 1000);

		setTimeout(() => {
			console.log('send shutdown request');
			agonesSDK.shutdown();
		}, 1000);
		setTimeout( () => {
			console.log('closing agones SDK');
			// Closing Agones SDK and all event emitters
			agonesSDK.close()
		}, 2000);
		setTimeout(() => {
			process.exit(0);
		}, 2000);
	} catch (error) {
		console.error(error);
	}
};

connect();
