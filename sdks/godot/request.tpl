{{$path := .path}}
{{range $index, $element := .data }}
{{$paramCount := MaxIndex $element.parameters}}

{{if $element.summary }}# {{ Replace $element.summary "\n" "\n# " -1}}{{end}}
func {{$element.operationId}}(
	{{range $i, $param := $element.parameters }}
	{{if ne (GetSchemRef $param.schema) "#/definitions/sdkEmpty"}}
	{{$param.name}} {{if $param.type}}: {{ ToGodotType $param.type}} {{end}}{{if ne $param.required true}}= ""{{end}}
	{{if ne $paramCount $i}},{{end}}
	{{end}}
	{{end}}
) -> Dictionary:
	return yield(_api_request({{ParseUrlParams $path $element.parameters}}, {
		{{range $i, $param := $element.parameters }}
			{{if and (ne (GetSchemRef $param.schema) "#/definitions/sdkEmpty") (eq $param.in "body")}}
			"{{$param.name}}" : {{$param.name}},
			{{end}}
		{{end}}
	 }, HTTPClient.METHOD_{{ ToUpper $index }}), "completed")
{{end}}