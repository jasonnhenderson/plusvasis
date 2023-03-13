package templates

import (
	"bytes"
	"text/template"
)

type NomadJob struct {
	ID    string `json:"id"`
	Name  string `json:"containerName"`
	Image string `json:"dockerImage"`
	User  string `json:"user"`
}

func CreateJobJson(job NomadJob) (*bytes.Buffer, error) {
	t, err := template.New("").Parse(JOB_TMPL)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, job)
	if err != nil {
		return nil, err
	}
	return buf, err
}

const JOB_TMPL = `{
	"Job": {
		"ID": "{{.ID}}",
		"Name": "{{.Name}}",
		"Type": "service",
		"Datacenters": [
			"dc1"
		],
        "Meta": {
            "user": "{{.User}}"
        },
		"TaskGroups": [
			{
				"Name": "{{.Name}}",
				"Count": 1,
				"Tasks": [
					{
						"Name": "{{.Name}}",
						"Driver": "docker",
						"Config": {
							"image": "{{.Image}}",
							"ports": [
								"http"
							]
						}
					}
				],
				"Networks": [
					{
						"Mode": "bridge",
						"DynamicPorts": [
							{
								"Label": "http",
								"To": 80
							}
						]
					}
				],
				"Services": [
					{
						"Name": "{{.Name}}",
						"PortLabel": "http",
						"Provider": "nomad"
					}
				]
			}
		]
	}
}`