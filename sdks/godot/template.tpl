extends HTTPRequest

{{ .header }}

signal on_request(path, params, method)

export(String) var api_endpoint = "http://localhost"

func _init():
	var agones_port = OS.get_environment("AGONES_SDK_HTTP_PORT")
	if !agones_port:
		agones_port = 9358
	api_endpoint = "http://127.0.0.1:%s" % agones_port

{{ .data }}

func _api_request(path : String, params : Dictionary, method = HTTPClient.METHOD_GET) -> Dictionary:

	emit_signal("on_request", path, params, method)

	# Build required request objects
	var request_string = "%s%s%s" % [api_endpoint, path, _params_to_string(params)]
	var headers : PoolStringArray = ["Content-Type: application/json"]
	
	# Make HTTP Requesst
	var error = self.request(request_string, headers, false, method)
	if error != OK:
		yield(get_tree().create_timer(0.001),"timeout")
		return AgonesError.new("Agones Client encounted an error Godot error code: %s" % error)

	# Get and parse result
	var result = yield(self, "request_completed")
	if len(result) > 3:
		if result[1] == 200:
			var json : JSONParseResult = JSON.parse(result[3].get_string_from_utf8())
			if json.error:
				return AgonesError.new("Failed to parse response: %s" % json.error_string)
			return json.result
		else: # Return response code in error message if possible
			return AgonesError.new("Request failed! Response code: %s\n%s" % [str(result[1]), str(result[3])])
		
	return AgonesError.new("Request failed!")

# Helper function for converting a dictionary into HTTP parameters
func _params_to_string(params : Dictionary) -> String:

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
