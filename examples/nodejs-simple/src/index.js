const AgonesSDK = require('agones');

let agonesSDK = new AgonesSDK();

let connect = async function() {
	agonesSDK.watchGameServer((result) => {
		console.log('watch', result);
	});

	try {
		await agonesSDK.ready();
		await agonesSDK.setLabel("label", "labelValue");
		await agonesSDK.setAnnotation("annotation", "annotationValue");
		let result = await agonesSDK.getGameServer();
		console.log('gameServer', result);
		setTimeout(() => {
			console.log('send health ping');
			agonesSDK.health();
		}, 2000);
		setTimeout(() => {
			console.log('send shutdown request');
			agonesSDK.shutdown();
		}, 4000);
		setTimeout(() => {
			agonesSDK.close();
		}, 6000);
	} catch (error) {
		console.error(error);
	}
};

connect();
