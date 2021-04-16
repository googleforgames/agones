extends HTTPRequest

# This code is generated by go generate.
# DO NOT EDIT BY HAND!

signal on_request(path, params, method)

export (String) var api_endpoint = "http://localhost"


func _init():
	var agones_port = OS.get_environment("AGONES_SDK_HTTP_PORT")
	if ! agones_port:
		agones_port = 9358
	api_endpoint = "http://127.0.0.1:%s" % agones_port


# Retrieves the current player capacity. This is always accurate from what has been set through this SDK,
# even if the value has yet to be updated on the GameServer status resource.
func GetPlayerCapacity() -> Dictionary:
	return yield(_api_request("/alpha/player/capacity", {}, HTTPClient.METHOD_GET), "completed")


# Update the GameServer.Status.Players.Capacity value with a new capacity.
func SetPlayerCapacity(body) -> Dictionary:
	return yield(
		_api_request(
			"/alpha/player/capacity",
			{
				"body": body,
			},
			HTTPClient.METHOD_PUT
		),
		"completed"
	)


# PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
func PlayerConnect(body) -> Dictionary:
	return yield(
		_api_request(
			"/alpha/player/connect",
			{
				"body": body,
			},
			HTTPClient.METHOD_POST
		),
		"completed"
	)


# Returns the list of the currently connected player ids. This is always accurate from what has been set through this SDK,
# even if the value has yet to be updated on the GameServer status resource.
func GetConnectedPlayers() -> Dictionary:
	return yield(_api_request("/alpha/player/connected", {}, HTTPClient.METHOD_GET), "completed")


# Returns if the playerID is currently connected to the GameServer. This is always accurate from what has been set through this SDK,
# even if the value has yet to be updated on the GameServer status resource.
func IsPlayerConnected(playerID: String) -> Dictionary:
	return yield(
		_api_request("/alpha/player/connected/%s" % [playerID], {}, HTTPClient.METHOD_GET),
		"completed"
	)


# Retrieves the current player count. This is always accurate from what has been set through this SDK,
# even if the value has yet to be updated on the GameServer status resource.
func GetPlayerCount() -> Dictionary:
	return yield(_api_request("/alpha/player/count", {}, HTTPClient.METHOD_GET), "completed")


# Decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
func PlayerDisconnect(body) -> Dictionary:
	return yield(
		_api_request(
			"/alpha/player/disconnect",
			{
				"body": body,
			},
			HTTPClient.METHOD_POST
		),
		"completed"
	)


func _api_request(path: String, params: Dictionary, method = HTTPClient.METHOD_GET) -> Dictionary:
	emit_signal("on_request", path, params, method)

	# Build required request objects
	var request_string = "%s%s%s" % [api_endpoint, path, _params_to_string(params)]
	var headers: PoolStringArray = ["Content-Type: application/json"]

	# Make HTTP Requesst
	var error = self.request(request_string, headers, false, method)
	if error != OK:
		yield(get_tree().create_timer(0.001), "timeout")
		return _build_error_message("Agones Client encounted an error Godot error code: %s" % error)

	# Get and parse result
	var result = yield(self, "request_completed")
	if len(result) > 3:
		if result[1] == 200:
			var json: JSONParseResult = JSON.parse(result[3].get_string_from_utf8())
			if json.error:
				return _build_error_message("Failed to parse response: %s" % json.error_string)
			return json.result
		else:  # Return response code in error message if possible
			return _build_error_message(
				"Request failed! Response code: %s\n%s" % [str(result[1]), str(result[3])]
			)

	return _build_error_message("Request failed!")


# Helper function for converting a dictionary into HTTP parameters
func _params_to_string(params: Dictionary) -> String:
	var param_strings = []
	for param in params:
		param_strings.append("%s=%s" % [param, str(params[param])])

	var params_string = ""
	for i in range(param_strings.size()):
		if i == 0:
			params_string += "?"

		params_string += param_strings[i]

		if i != params.size():
			params_string += "&"
	return params_string


# Helper function for generating client errors
func _build_error_message(message):
	return {"message": message, "success": false}
