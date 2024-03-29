package nomad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"plusvasis/internal/templates"

	nomad "github.com/hashicorp/nomad/nomad/structs"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type NomadController struct {
	Client NomadClient
}

// GetJobs godoc
//
//	@Summary		GetJobs
//	@Description	Get details of all Nomad jobs
//	@Tags			nomad
//	@Produce		json
//	@Success		200	{object}	[]nomad.JobListStub
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/jobs [get]
func (n *NomadController) GetJobs(c echo.Context) error {
	data, err := n.Client.Get("/jobs?meta=true")
	if err != nil {
		return err
	}

	var jobs []nomad.JobListStub
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		return echo.ErrInternalServerError
	}

	var filteredJobs []nomad.JobListStub
	uid := c.Get("uid").(string)
	for _, job := range jobs {
		if job.Meta["user"] == uid {
			filteredJobs = append(filteredJobs, job)
		}
	}

	return c.JSON(http.StatusOK, filteredJobs)
}

// CreateJob godoc
//
//	@Summary		CreateJob
//	@Description	Create a new Nomad job
//	@Tags			nomad
//	@Accept			json
//	@Produce		json
//	@Param			job	body		templates.NomadJob	true	"Nomad Job"
//	@Success		200	{object}	nomad.JobRegisterResponse
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/jobs [post]
func (n *NomadController) CreateJob(c echo.Context) error {
	var job templates.NomadJob
	err := decodeJobJson(&job, c.Request().Body)
	if err != nil {
		return err
	}

	body, err := templates.CreateJobJson(job)
	if err != nil {
		return errors.Wrap(echo.ErrBadRequest, err.Error())
	}

	data, err := n.Client.Post("/jobs", body)
	if err != nil {
		return err
	}

	var resp nomad.JobRegisterResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateJob godoc
//
//	@Summary		UpdateJob
//	@Description	Update an existing Nomad job
//	@Tags			nomad
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"Job ID"
//	@Param			job	body		templates.NomadJob	true	"Nomad Job"
//	@Success		200	{object}	nomad.JobRegisterResponse
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/job/{id} [post]
func (n *NomadController) UpdateJob(c echo.Context) error {
	var job templates.NomadJob
	err := decodeJobJson(&job, c.Request().Body)
	if err != nil {
		return err
	}

	body, err := templates.CreateJobJson(job)
	if err != nil {
		return errors.Wrap(echo.ErrBadRequest, err.Error())
	}

	uid := c.Get("uid").(string)
	jobId := c.Param("id")
	if err := n.CheckUserAllowed(uid, jobId); err != nil {
		return err
	}

	data, err := n.Client.Post(fmt.Sprintf("/job/%s", jobId), body)
	if err != nil {
		return err
	}

	var resp nomad.JobRegisterResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

// ReadJob godoc
//
//	@Summary		ReadJob
//	@Description	Get details of a Nomad job
//	@Tags			nomad
//	@Produce		json
//	@Param			id	path		string	true	"Job ID"
//	@Success		200	{object}	nomad.JobListStub
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/job/{id} [get]
func (n *NomadController) ReadJob(c echo.Context) error {
	uid := c.Get("uid").(string)
	jobId := c.Param("id")

	data, err := n.Client.Get(fmt.Sprintf("/job/%s", jobId))
	if err != nil {
		return err
	}

	var job nomad.Job
	err = json.Unmarshal(data, &job)
	if err != nil {
		return echo.ErrInternalServerError
	}

	// Doing this check here because if we use n.CheckUserAllowed, we will duplicate requests
	if job.Meta["user"] != uid {
		return echo.ErrUnauthorized
	}

	return c.JSON(http.StatusOK, job)
}

// StopJob godoc
//
//	@Summary		StopJob
//	@Description	Stop a running Nomad job
//	@Tags			nomad
//	@Produce		json
//	@Param			id		path		string	true	"Job ID"
//	@Param			purge	query		string	false	"Purge job"
//	@Success		200		{object}	nomad.JobDeregisterResponse
//	@Failure		401		{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/job/{id} [delete]
func (n *NomadController) StopJob(c echo.Context) error {
	uid := c.Get("uid").(string)
	jobId := c.Param("id")
	purge := c.QueryParam("purge")

	if err := n.CheckUserAllowed(uid, jobId); err != nil {
		return err
	}

	url := fmt.Sprintf("/job/%s", jobId)
	if purge == "true" {
		url += "?purge=true"
	}
	data, err := n.Client.Delete(url)
	if err != nil {
		return err
	}

	var resp nomad.JobDeregisterResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

// RestartJob godoc
//
//	@Summary		RestartJob
//	@Description	Restart a Nomad job
//	@Tags			nomad
//	@Produce		json
//	@Param			id	path		string	true	"Job ID"
//	@Success		200	{object}	nomad.GenericResponse
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/job/{id}/restart [post]
func (n *NomadController) RestartJob(c echo.Context) error {
	uid := c.Get("uid").(string)
	jobId := c.Param("id")

	if err := n.CheckUserAllowed(uid, jobId); err != nil {
		return err
	}

	alloc, err := n.ParseRunningAlloc(jobId)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer([]byte{})
	data, err := n.Client.Post(fmt.Sprintf("/client/allocation/%s/restart", alloc.ID), body)
	if err != nil {
		return err
	}

	var resp nomad.GenericResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

// StartJob godoc
//
//	@Summary		StartJob
//	@Description	Start a stopped Nomad job
//	@Tags			nomad
//	@Produce		json
//	@Param			id	path		string	true	"Job ID"
//	@Success		200	{object}	nomad.JobRegisterResponse
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/job/{id}/start [get]
func (n *NomadController) StartJob(c echo.Context) error {
	uid := c.Get("uid").(string)
	jobId := c.Param("id")

	data, err := n.Client.Get(fmt.Sprintf("/job/%s", jobId))
	if err != nil {
		return err
	}

	var job nomad.Job
	err = json.Unmarshal(data, &job)
	if err != nil {
		return err
	}

	if job.Meta["user"] != uid {
		return echo.ErrUnauthorized
	}

	// Nomad doesn't have a start job endpoint, and this
	// is exactly how they do it in their Web UI
	// It's a bit hacky, but it works
	job.Stop = false
	var jobRequest nomad.JobRegisterRequest
	jobRequest.Job = &job

	body, err := json.Marshal(jobRequest)
	if err != nil {
		return echo.ErrInternalServerError
	}

	data, err = n.Client.Post(fmt.Sprintf("/job/%s", jobId), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	var resp nomad.JobRegisterResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func (n *NomadController) CheckUserAllowed(uid, jobId string) error {
	data, err := n.Client.Get(fmt.Sprintf("/job/%s", jobId))
	if err != nil {
		return err
	}

	var job nomad.Job
	err = json.Unmarshal(data, &job)
	if err != nil {
		return echo.ErrInternalServerError
	}

	if job.Meta["user"] != uid {
		return echo.ErrUnauthorized
	}

	return nil
}

func (n *NomadController) ParseRunningAlloc(jobId string) (*nomad.AllocListStub, error) {
	data, err := n.Client.Get(fmt.Sprintf("/job/%s/allocations", jobId))
	if err != nil {
		return nil, err
	}

	var allocs []nomad.AllocListStub
	err = json.Unmarshal(data, &allocs)
	if err != nil {
		return nil, echo.ErrInternalServerError
	}
	for _, alloc := range allocs {
		if alloc.ClientStatus == "running" || alloc.ClientStatus == "pending" {
			return &alloc, nil
		}
	}

	return nil, echo.ErrNotFound
}
