{{ define "packages" }}

{{ with .packages}}
<p>Packages:</p>
<ul>
    {{ range . }}
    <li>
        <a href="#{{- packageDisplayName . -}}">{{ packageDisplayName . }}</a>
    </li>
    {{ end }}
</ul>
{{ end}}

{{ range .packages }}
    <h2 id="{{- packageDisplayName . -}}">
        {{- packageDisplayName . -}}
    </h2>

    {{ with .DocComments }}
    <p>
        {{ safe (renderComments .) }}
    </p>
    {{ end }}

    Resource Types:
    <ul>
    {{- range (visibleTypes (sortedTypes .Types)) -}}
        {{ if isExportedType . -}}
        <li>
            <a href="#{{ typeIdentifier . }}">{{ typeIdentifier . }}</a>
        </li>
        {{- end }}
    {{- end -}}
    </ul>

    {{ range (visibleTypes (sortedTypes .Types))}}
        {{ template "type" .  }}
    {{ end }}
    <hr/>
{{ end }}

<p><em>
    Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>

{{ end }}
