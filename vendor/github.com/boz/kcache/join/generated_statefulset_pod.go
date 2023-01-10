/*
 * AUTO GENERATED - DO NOT EDIT BY HAND
 */

package join

import (
	"context"

	logutil "github.com/boz/go-logutil"
	"github.com/boz/kcache/filter"
	"github.com/boz/kcache/types/pod"
	"github.com/boz/kcache/types/statefulset"
	appsv1 "k8s.io/api/apps/v1"
)

func StatefulSetPodsWith(ctx context.Context,
	srcController statefulset.Controller,
	dstController pod.Publisher,
	filterFn func(...*appsv1.StatefulSet) filter.ComparableFilter) (pod.Controller, error) {

	log := logutil.FromContextOrDefault(ctx)

	dst, err := dstController.CloneForFilter()
	if err != nil {
		return nil, err
	}

	update := func(_ *appsv1.StatefulSet) {
		objs, err := srcController.Cache().List()
		if err != nil {
			log.Err(err, "join(statefulset,pod: cache list")
			return
		}
		dst.Refilter(filterFn(objs...))
	}

	handler := statefulset.BuildHandler().
		OnInitialize(func(objs []*appsv1.StatefulSet) { dst.Refilter(filterFn(objs...)) }).
		OnCreate(update).
		OnUpdate(update).
		OnDelete(update).
		Create()

	monitor, err := statefulset.NewMonitor(srcController, handler)
	if err != nil {
		dst.Close()
		return nil, log.Err(err, "join(statefulset,pod): monitor")
	}

	go func() {
		<-dst.Done()
		monitor.Close()
	}()

	return dst, nil
}
