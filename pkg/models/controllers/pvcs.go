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

package controllers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (ctl *PvcCtl) generateObject(item *v1.PersistentVolumeClaim) *Pvc {
	name := item.Name
	namespace := item.Namespace
	status := fmt.Sprintf("%s", item.Status.Phase)
	createTime := item.CreationTimestamp.Time
	var capacity, storageClass, accessModeStr string

	if createTime.IsZero() {
		createTime = time.Now()
	}

	if storage, exist := item.Status.Capacity["storage"]; exist {
		capacity = storage.String()
	}

	if len(item.Annotations["volume.beta.kubernetes.io/storage-class"]) > 0 {
		storageClass = item.Annotations["volume.beta.kubernetes.io/storage-class"]
	}
	if item.Spec.StorageClassName != nil {
		storageClass = *item.Spec.StorageClassName
	}

	var accessModeList []string
	for _, accessMode := range item.Status.AccessModes {
		accessModeList = append(accessModeList, string(accessMode))
	}

	accessModeStr = strings.Join(accessModeList, ",")

	object := &Pvc{Namespace: namespace, Name: name, Status: status, Capacity: capacity,
		AccessMode: accessModeStr, StorageClassName: storageClass, CreateTime: createTime, Annotation: Annotation{item.Annotations}}

	return object
}

func (ctl *PvcCtl) listAndWatch() {
	defer func() {
		close(ctl.aliveChan)
		if err := recover(); err != nil {
			glog.Error(err)
			return
		}
	}()

	db := ctl.DB

	if db.HasTable(&Pvc{}) {
		db.DropTable(&Pvc{})

	}

	db = db.CreateTable(&Pvc{})

	k8sClient := ctl.K8sClient
	kubeInformerFactory := informers.NewSharedInformerFactory(k8sClient, time.Second*resyncCircle)
	informer := kubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer()
	lister := kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister()

	list, err := lister.List(labels.Everything())
	if err != nil {
		glog.Error(err)
		return
	}

	for _, item := range list {
		obj := ctl.generateObject(item)
		db.Create(obj)

	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {

			object := obj.(*v1.PersistentVolumeClaim)
			mysqlObject := ctl.generateObject(object)
			db.Create(mysqlObject)
		},
		UpdateFunc: func(old, new interface{}) {
			object := new.(*v1.PersistentVolumeClaim)
			mysqlObject := ctl.generateObject(object)
			db.Save(mysqlObject)
		},
		DeleteFunc: func(obj interface{}) {
			var item Pvc
			object := obj.(*v1.PersistentVolumeClaim)
			db.Where("name=? And namespace=?", object.Name, object.Namespace).Find(&item)
			db.Delete(item)
		},
	})

	informer.Run(ctl.stopChan)
}

func (ctl *PvcCtl) CountWithConditions(conditions string) int {
	var object Pvc

	return countWithConditions(ctl.DB, conditions, &object)
}

func (ctl *PvcCtl) ListWithConditions(conditions string, paging *Paging) (int, interface{}, error) {
	var list []Pvc
	var object Pvc
	var total int

	order := "createTime desc"

	listWithConditions(ctl.DB, &total, &object, &list, conditions, paging, order)

	for index := range list {
		inUsePods := list[index].Annotation.Values[inUse]
		var pods []string

		json.Unmarshal([]byte(inUsePods), &pods)

		if len(pods) > 0 {
			list[index].InUse = true
		} else {
			list[index].InUse = false
		}
	}

	return total, list, nil
}

func (ctl *PvcCtl) Count(namespace string) int {
	var count int
	db := ctl.DB
	if len(namespace) == 0 {
		db.Model(&Pvc{}).Count(&count)
	} else {
		db.Model(&Pvc{}).Where("namespace = ?", namespace).Count(&count)
	}
	return count
}
