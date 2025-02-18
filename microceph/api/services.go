package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/canonical/microceph/microceph/api/types"
	"github.com/canonical/microceph/microceph/common"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microceph/microceph/ceph"
)

// /1.0/services endpoint.
var servicesCmd = rest.Endpoint{
	Path: "services",

	Get: rest.EndpointAction{Handler: cmdServicesGet, ProxyTarget: true},
}

func cmdServicesGet(s *state.State, r *http.Request) response.Response {
	services, err := ceph.ListServices(s)
	if err != nil {
		return response.InternalError(err)
	}

	return response.SyncResponse(true, services)
}

// Service Enable Endpoint.
var monServiceCmd = rest.Endpoint{
	Path: "services/mon",
	Put:  rest.EndpointAction{Handler: cmdEnableServicePut, ProxyTarget: true},
}

var mgrServiceCmd = rest.Endpoint{
	Path: "services/mgr",
	Put:  rest.EndpointAction{Handler: cmdEnableServicePut, ProxyTarget: true},
}

var mdsServiceCmd = rest.Endpoint{
	Path: "services/mds",
	Put:  rest.EndpointAction{Handler: cmdEnableServicePut, ProxyTarget: true},
}

var rgwServiceCmd = rest.Endpoint{
	Path:   "services/rgw",
	Put:    rest.EndpointAction{Handler: cmdEnableServicePut, ProxyTarget: true},
	Delete: rest.EndpointAction{Handler: cmdRGWServiceDelete, ProxyTarget: true},
}

func cmdEnableServicePut(s *state.State, r *http.Request) response.Response {
	var payload types.EnableService

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		logger.Errorf("Failed decoding enable service request: %v", err)
		return response.InternalError(err)
	}

	err = ceph.ServicePlacementHandler(common.CephState{State: s}, payload)
	if err != nil {
		return response.SyncResponse(false, err)
	}

	return response.SyncResponse(true, nil)
}

// Service Reload Endpoint.
var restartServiceCmd = rest.Endpoint{
	Path: "services/restart",
	Post: rest.EndpointAction{Handler: cmdRestartServicePost, ProxyTarget: true},
}

func cmdRestartServicePost(s *state.State, r *http.Request) response.Response {
	var services types.Services

	err := json.NewDecoder(r.Body).Decode(&services)
	if err != nil {
		logger.Errorf("Failed decoding restart services: %v", err)
		return response.InternalError(err)
	}

	// Check if provided services are valid and available in microceph
	for _, service := range services {
		valid_services := ceph.GetConfigTableServiceSet()
		if _, ok := valid_services[service.Service]; !ok {
			err := fmt.Errorf("%s is not a valid ceph service", service.Service)
			logger.Errorf("%v", err)
			return response.InternalError(err)
		}
	}

	for _, service := range services {
		err = ceph.RestartCephService(service.Service)
		if err != nil {
			url := s.Address().String()
			logger.Errorf("Failed restarting %s on host %s", service.Service, url)
			return response.SyncResponse(false, err)
		}
	}

	return response.EmptySyncResponse
}

func cmdRGWServiceDelete(s *state.State, r *http.Request) response.Response {
	var req types.RGWService

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return response.InternalError(err)
	}

	err = ceph.DisableRGW(common.CephState{State: s})
	if err != nil {
		return response.SmartError(err)
	}

	return response.EmptySyncResponse
}
