/*
Copyright 2018 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"net/http"

	"github.com/emicklei/go-restful"

	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/models"
)

func Register(ws *restful.WebService, subPath string) {

	ws.Route(ws.GET(subPath + "/{resource}").To(listResource).Produces(restful.MIME_JSON))

}

func listResource(req *restful.Request, resp *restful.Response) {

	resource := req.PathParameter("resource")
	conditions := req.QueryParameter("conditions")
	paging := req.QueryParameter("paging")

	res, err := models.ListResource(resource, conditions, paging)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, constants.MessageResponse{Message: err.Error()})
		return
	}

	resp.WriteEntity(res)
}
