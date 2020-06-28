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

const AlphaAgonesSDK = require('@google-cloud/agones-sdk').alphaSDK;
const util = require('util');

const sleep = util.promisify(setTimeout);

const agonesSDK = new AlphaAgonesSDK();

const connect = async () => {
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
			console.log(`GameServer Update:
	name: ${result.objectMeta.name}
	state: ${result.status.state}
	labels: ${result.objectMeta.labelsMap.join(' & ')}
	annotations: ${result.objectMeta.annotationsMap.join(' & ')}
	players: ${result.status.players.count}/${result.status.players.capacity} [${result.status.players.idsList}]`);
		});

		await sleep(10000);
		console.log('Setting capacity');
		await agonesSDK.setPlayerCapacity(64);

		await sleep(10000);
		console.log('Getting capacity');
		let result = await agonesSDK.getPlayerCapacity();
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Connecting a player');
		result = await agonesSDK.playerConnect('firstPlayerID');
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Connecting a duplicate player');
		result = await agonesSDK.playerConnect('firstPlayerID');
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Connecting another player');
		await agonesSDK.playerConnect('secondPlayerID');

		await sleep(10000);
		console.log('Getting player count');
		result = await agonesSDK.getPlayerCount();
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Finding if firstPlayerID connected');
		result = await agonesSDK.isPlayerConnected('firstPlayerID');
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Getting connected players');
		result = await agonesSDK.getConnectedPlayers();
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Disconnecting a player');
		result = await agonesSDK.playerDisconnect('firstPlayerID');
		console.log(`result: ${result}`);

		await sleep(10000);
		console.log('Disconnecting the same player');
		result = await agonesSDK.playerDisconnect('firstPlayerID');
		console.log(`result: ${result}`);

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
		process.exit(0);
	}
};

let args = process.argv.slice(2);

for (let arg of args) {
	let [argName, argValue] = arg.split('=');
	if (argName === '--help') {
		console.log(`Example to call each alpha SDK feature in turn. Once complete will call shutdown and close.`);
		return;
	}
}

connect();
