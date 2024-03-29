package templates

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

type NomadJob struct {
	Name      string     `json:"containerName" validate:"required"`
	Image     string     `json:"dockerImage" validate:"required"`
	User      string     `json:"user" validate:"required"`
	Shell     string     `json:"shell" validate:"required"`
	Volumes   [][]string `json:"volumes" validate:"kvPairs"`
	Env       [][]string `json:"env" validate:"kvPairs"`
	EnvString string     `json:"envString"`
	Port      int        `json:"port" validate:"min=0,max=65535"`
	Expose    bool       `json:"expose"`
	Cpu       int        `json:"cpu" validate:"min=0,max=1000"`
	Memory    int        `json:"memory" validate:"min=0,max=2000"`
}

func last(i int, slice interface{}) bool {
	v := reflect.ValueOf(slice)
	return i == v.Len()-1
}

func CreateJobJson(job NomadJob) (*bytes.Buffer, error) {
	// create template
	t, err := template.New("").Funcs(template.FuncMap{
		"last": last,
	}).Parse(JOB_TMPL)
	if err != nil {
		return nil, err
	}

	// ensure job is valid
	err = Validate(job)
	if err != nil {
		return nil, err
	}

	// parse env vars for templating
	if len(job.Env) != 0 {
		trimEnv(&job)
		err = parseEnv(&job)
		if err != nil {
			return nil, err
		}
	}

	// execute template
	buf := &bytes.Buffer{}
	err = t.Execute(buf, job)
	if err != nil {
		return nil, err
	}

	// output for debugging
	f, err := os.Create("latest-job.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return buf, err
}

func trimEnv(job *NomadJob) {
	for i, env := range job.Env {
		// trim spaces from key and value
		job.Env[i][0] = strings.TrimSpace(env[0])
		job.Env[i][1] = strings.TrimSpace(env[1])

		// trim either single or double quotes from value
		if strings.HasPrefix(env[1], "'") && strings.HasSuffix(env[1], "'") {
			job.Env[i][1] = strings.Trim(env[1], "'")
		} else if strings.HasPrefix(env[1], "\"") && strings.HasSuffix(env[1], "\"") {
			job.Env[i][1] = strings.Trim(env[1], "\"")
		}
	}
}

func parseEnv(job *NomadJob) error {
	for _, env := range job.Env {
		key := env[0]
		value := env[1]

		fmt.Printf("key: %s, value: %s\n", key, value)

		fieldRegex := regexp.MustCompile(`{{\s*(.*?)\s*}}`)
		fields := fieldRegex.FindAllStringSubmatch(value, -1)

		fmt.Printf("fields: %v\n", fields)
		if len(fields) > 0 {
			// ensure fields[i][1] are all equal
			fieldVal := fields[0][1]
			for _, field := range fields {
				if field[1] != fieldVal {
					return fmt.Errorf("only one other job can be referenced in a templated env var")
				}
			}

			job.EnvString += generateTemplatedEnv(key, value, fieldVal)
		}
	}
	return nil
}

func generateTemplatedEnv(key, value, otherJob string) string {
	// replace every instance of otherJob
	newValue := strings.ReplaceAll(value, fmt.Sprintf("{{%s}}", otherJob), "{{ .Address }}:{{ .Port }}")
	templatedEnv := fmt.Sprintf(ENV_TMPL, otherJob, key, newValue)

	// escape newline characters and double quotes
	// so that the template can be embedded in a JSON string
	templatedEnv = strings.ReplaceAll(templatedEnv, "\n", "\\n")
	templatedEnv = strings.ReplaceAll(templatedEnv, "\"", "\\\"")

	return templatedEnv
}

const ENV_TMPL = `{{ range nomadService "%s" }}
%s="%s"
{{ end }}
`

const JOB_TMPL = `{
    "Job": {
        "ID": "{{.User}}-{{.Name}}",
        "Name": "{{.Name}}",
        "Type": "service",
        "Datacenters": [
            "dc1"
        ],
        "Meta": {
            "user": "{{.User}}",
            "shell": "{{.Shell}}",
            "volumes": "{{range $i, $v := .Volumes}}{{index $v 0}}:{{index $v 1}}{{if not (last $i $.Volumes)}},{{end}}{{end}}",
            "env": "{{range $i, $v := .Env}}{{index $v 0}}={{index $v 1}}{{if not (last $i $.Env)}},{{end}}{{end}}",
            "port": "{{.Port}}",
            "expose": "{{.Expose}}"
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
                                "port"
                            ],
                            "mount": [
                                {{range $_, $v := .Volumes}}
                                {
                                    "type": "volume",
                                    "readonly": false,
                                    "source": "plusvasis-{{$.User}}-{{index $v 0}}",
                                    "target": "{{index $v 1}}"
                                },
                                {{end}}
                                {
                                    "type": "volume",
                                    "readonly": false,
                                    "source": "plusvasis-{{.User}}",
                                    "target": "/userdata"
                                }
                            ]
                        },
                        "Resources": {
                            "CPU": {{.Cpu}},
                            "MemoryMB": {{.Memory}}
                        }
                        {{if .Env}},
                        "Env": {
                            {{range $i, $v := .Env}}
                            "{{index $v 0}}": "{{index $v 1}}"{{if not (last $i $.Env)}},{{end}}
                            {{end}}
                        }
                        {{end}}
                        {{if .EnvString}},
                        "Templates": [
                            {
                                "DestPath": "secrets/config.env",
                                "EmbeddedTmpl": "{{.EnvString}}",
                                "Envvars": true
                            }
                        ]
                        {{end}}
                    }
                ],
                "Networks": [
                    {
                        "Mode": "host",
                        "DynamicPorts": [
                            {
                                "Label": "port",
                                "To": {{.Port}}
                            }
                        ]
                    }
                ],
                "Services": [
                    {
                        "Name": "{{.Name}}",
                        "PortLabel": "port",
                        {{if .Expose}}
                        "Tags": [
                            "traefik.enable=true",
                            "traefik.http.routers.{{.User}}-{{.Name}}.entrypoints=https",
                            "traefik.http.routers.{{.User}}-{{.Name}}.rule=Host(` + "`" + `{{.User}}-{{.Name}}.plusvasis.xyz` + "`" + `)",
                            "traefik.port=${NOMAD_PORT_port}"
                        ],
                        {{end}}
                        "Provider": "nomad"
                    }
                ]
            }
        ]
    }
}`
